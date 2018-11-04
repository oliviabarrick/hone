package kubernetes


import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/justinbarrick/farm/pkg/storage"
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/job"
	"github.com/justinbarrick/farm/pkg/logger"
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
	Namespace *string `hcl:"namespace"`
	Kubeconfig *string `hcl:"kubeconfig"`
}

func (k *Kubernetes) Run(c cache.Cache, j *job.Job) error {
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

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	storageCacheKey, err := storage.UploadInputs(c, j)
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
			Value: storageCacheKey,
		},
		{
			Name: "OUTPUTS",
			Value: string(outputs),
		},
		{
			Name: "CA_FILE",
			Value: "/build/.farm-ca-certificates.crt",
		},
	}

	if j.Env != nil {
		for name, value := range *j.Env {
			env = append(env, corev1.EnvVar{
				Name:  name,
				Value: value,
			})
		}
	}

	cacheEnv := c.Env()

	namespace := "default"
	if k.Namespace != nil {
		namespace = *k.Namespace
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Create(&corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"farm/target": j.Name,
			},
		},
		StringData: cacheEnv,
	})
	if err != nil {
		return err
	}
	defer clientset.CoreV1().Secrets(namespace).Delete(secret.Name, &metav1.DeleteOptions{})

	for key, _ := range cacheEnv {
		env = append(env, corev1.EnvVar{
			Name: key,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secret.Name},
					Key: key,
				},
			},
		})
	}

	pod, err := clientset.CoreV1().Pods(namespace).Create(&corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"farm/target": j.Name,
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
					Name: "cache-shim",
					Image: "justinbarrick/cache-shim",
					ImagePullPolicy: "Always",
					Command: []string{"/bin/sh", "-c", "cp /cache-shim /build && cp /etc/ssl/certs/ca-certificates.crt /build/.farm-ca-certificates.crt",},
					WorkingDir: "/build",
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
					Name:            j.Name,
					Image:           j.Image,
					ImagePullPolicy: "IfNotPresent",
					Command: []string{
						"/build/cache-shim", "/bin/sh", "-cex", j.Shell,
					},
					WorkingDir: "/build",
					Env:        env,
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
	defer clientset.CoreV1().Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{})

	watcher, err := clientset.CoreV1().Pods(namespace).Watch(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("farm/target=%s", j.Name),
	})
	if err != nil {
		return err
	}

	watchCh := watcher.ResultChan()
	running := false

	for event := range watchCh {
		pod = event.Object.(*corev1.Pod)

		if pod.Status.Phase == "Running" {
			running = true
		}

		if pod.Status.Phase != "Pending" && pod.Status.Phase != "PodInitializing" {
			break
		}
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
  	Container: j.Name,
		Follow: true,      
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
	if _, err = cache.LoadCache(c, storageCacheKey, j); err != nil {
		return err
	}

	return nil
}
