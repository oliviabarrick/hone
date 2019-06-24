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
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Kubernetes struct {
	Namespace  *string `hcl:"namespace"`
	Cache      cache.Cache
	cacheKey   string
	pod        string
	watchCh    <-chan watch.Event
	clientset  *kubernetes.Clientset
}

func (k *Kubernetes) Init() error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	k.clientset, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	if k.Namespace == nil {
		namespace := "default"
		k.Namespace = &namespace
	}

	return nil
}

func (k *Kubernetes) watch(j *job.Job) (<-chan watch.Event, error) {
	if k.watchCh != nil {
		return k.watchCh, nil
	}

	watcher, err := k.clientset.CoreV1().Pods(*k.Namespace).Watch(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("hone/target=%s", j.GetName()),
	})
	if err != nil {
		return nil, err
	}

	watchCh := watcher.ResultChan()

	var pod *corev1.Pod

	for event := range watchCh {
		pod = event.Object.(*corev1.Pod)

		if pod.Status.Phase != "Pending" && pod.Status.Phase != "PodInitializing" {
			break
		}
	}

	k.watchCh = watchCh

	if pod.Status.Phase != "Running" && pod.Status.Phase != "Succeeded" {
		k.Logs(j, j.GetName())
		return k.watchCh, errors.New(fmt.Sprintf("Invalid pod status: %s", pod.Status.Phase))
	}

	return k.watchCh, nil
}

func (k *Kubernetes) Logs(j *job.Job, container string) error {
	req := k.clientset.CoreV1().Pods(*k.Namespace).GetLogs(k.pod, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
	})

	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	if _, err := io.Copy(logger.LogWriter(j), readCloser); err != nil {
		return err
	}

	return nil
}

func (k *Kubernetes) Wait(ctx context.Context, j *job.Job) error {
	if err := k.Logs(j, j.GetName()); err != nil {
		return err
	}

	watchCh, err := k.watch(j)
	if err != nil {
		return err
	}

	var pod *corev1.Pod

	for event := range watchCh {
		pod = event.Object.(*corev1.Pod)

		if pod.Status.Phase != "Running" {
			break
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
	k.clientset.CoreV1().Services(*k.Namespace).Delete(j.GetName(), &metav1.DeleteOptions{})
	k.clientset.CoreV1().Pods(*k.Namespace).Delete(k.pod, &metav1.DeleteOptions{})
	return nil
}

func (k *Kubernetes) Start(ctx context.Context, j *job.Job) error {
	var err error

	k.cacheKey, err = storage.UploadInputs(k.Cache, j)
	if err != nil {
		return err
	}

	outputs, err := json.Marshal(j.GetOutputs())
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

	_, err = k.clientset.CoreV1().Services(*k.Namespace).Create(&corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.GetName(),
			Namespace: *k.Namespace,
			Labels: map[string]string{
				"hone/target": j.GetName(),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"hone/target": j.GetName(),
			},
			ClusterIP: "None",
		},
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

	privileged := j.IsPrivileged()

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
					Command:         []string{
						"/bin/sh", "-c",
						"cp /cache-shim /build && cp /etc/ssl/certs/ca-certificates.crt /build/.hone-ca-certificates.crt",
					},
					WorkingDir:      filepath.Join("/build", j.GetWorkdir()),
					Env:             env,
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
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
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

	if _, err := k.watch(j); err != nil {
		return err
	}

	return nil
}
