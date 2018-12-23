package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/justinbarrick/hone/pkg/storage"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/user"
	"path/filepath"
)

type Kubernetes struct {
	Namespace  *string `hcl:"namespace"`
	Kubeconfig *string `hcl:"kubeconfig"`
	Cache      cache.Cache
	cacheKey   string
	clientset  *kubernetes.Clientset
	pod        string
}

func (k *Kubernetes) Init() error {
	kubeconfig := os.Getenv("KUBECONFIG")

	if k.Kubeconfig != nil {
		kubeconfig = *k.Kubeconfig
	}

	if kubeconfig == "" {
		usr, err := user.Current()
		if err != nil {
			return err
		}
		kubeconfig = filepath.Join(usr.HomeDir, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	k.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	if k.Namespace == nil {
		namespace := "default"
		k.Namespace = &namespace
	}

	return nil
}

func (k *Kubernetes) Wait(ctx context.Context, j *job.Job) error {
	watcher, err := k.clientset.CoreV1().Pods(*k.Namespace).Watch(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("hone/target=%s", j.GetName()),
	})
	if err != nil {
		return err
	}

	watchCh := watcher.ResultChan()
	running := false

	var pod *corev1.Pod

	for event := range watchCh {
		pod = event.Object.(*corev1.Pod)

		if pod.Status.Phase == "Running" {
			running = true
		}

		if pod.Status.Phase != "Pending" && pod.Status.Phase != "PodInitializing" {
			break
		}
	}

	req := k.clientset.CoreV1().Pods(*k.Namespace).GetLogs(k.pod, &corev1.PodLogOptions{
		Container: j.GetName(),
		Follow:    true,
	})

	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	io.Copy(logger.LogWriter(j), readCloser)

	if running {
		for event := range watchCh {
			pod = event.Object.(*corev1.Pod)

			if pod.Status.Phase != "Running" {
				break
			}
		}
	}

	exitStatus := pod.Status.ContainerStatuses[0].State.Terminated.ExitCode
	if exitStatus != 0 {
		return errors.New(fmt.Sprintf("Pod exited with error: %d", exitStatus))
	}

	logger.Log(j, fmt.Sprintf("Pod exit status %d, phase %s", exitStatus, pod.Status.Phase))

	if _, err = cache.LoadCache(k.Cache, k.cacheKey, j); err != nil {
		return err
	}

	return nil
}

func (k *Kubernetes) Stop(ctx context.Context, j *job.Job) error {
	k.clientset.CoreV1().Secrets(*k.Namespace).Delete(j.GetName(), &metav1.DeleteOptions{})
	k.clientset.CoreV1().Pods(*k.Namespace).Delete(k.pod, &metav1.DeleteOptions{})
	return nil
}

func (k *Kubernetes) Start(ctx context.Context, j *job.Job) error {
	var err error

	k.cacheKey, err = storage.UploadInputs(k.Cache, j)
	if err != nil {
		return err
	}

	outputs, err := json.Marshal(j.Outputs)
	if err != nil {
		return err
	}

	env := []corev1.EnvVar{
		{
			Name:  "CACHE_KEY",
			Value: k.cacheKey,
		},
		{
			Name:  "OUTPUTS",
			Value: string(outputs),
		},
		{
			Name:  "CA_FILE",
			Value: "/build/.hone-ca-certificates.crt",
		},
	}

	for name, value := range j.GetEnv() {
		env = append(env, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}

	cacheEnv := k.Cache.Env()

	secret, err := k.clientset.CoreV1().Secrets(*k.Namespace).Create(&corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.GetName(),
			Namespace: *k.Namespace,
			Labels: map[string]string{
				"hone/target": j.GetName(),
			},
		},
		StringData: cacheEnv,
	})
	if err != nil {
		return err
	}

	for key, _ := range cacheEnv {
		env = append(env, corev1.EnvVar{
			Name: key,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secret.Name},
					Key:                  key,
				},
			},
		})
	}

	cmdLine := []string{"/build/cache-shim"}
	cmdLine = append(cmdLine, j.GetShell()...)

	pod, err := k.clientset.CoreV1().Pods(*k.Namespace).Create(&corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.GetName(),
			Namespace: *k.Namespace,
			Labels: map[string]string{
				"hone/target": j.GetName(),
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "share",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: "Memory",
						},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:            "cache-shim",
					Image:           "justinbarrick/cache-shim",
					ImagePullPolicy: "Always",
					Command:         []string{"/bin/sh", "-c", "cp /cache-shim /build && cp /etc/ssl/certs/ca-certificates.crt /build/.hone-ca-certificates.crt"},
					WorkingDir:      "/build",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "share",
							MountPath: "/build",
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:            j.GetName(),
					Image:           j.GetImage(),
					ImagePullPolicy: "Always",
					Command:         cmdLine,
					WorkingDir:      "/build",
					Env:             env,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "share",
							MountPath: "/build",
						},
					},
				},
			},
			RestartPolicy: "Never",
		},
	})
	if err != nil {
		return err
	}

	k.pod = pod.Name
	return nil
}
