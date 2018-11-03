package kubernetes

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/justinbarrick/farm/pkg/job"
	"github.com/justinbarrick/farm/pkg/logger"
)

func Run(j job.Job) error {
	kubeconfig := filepath.Join("/Users/justin", ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	env := []corev1.EnvVar{}
	if j.Env != nil {
		for name, value := range *j.Env {
			env = append(env, corev1.EnvVar{
				Name: name,
				Value: value,
			})
		}
	}

	pod, err := clientset.CoreV1().Pods("u-jbarrick").Create(&corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: j.Name,
			Namespace: "u-jbarrick",
			Labels: map[string]string{
				"farm/target": j.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            j.Name,
					Image:           j.Image,
					ImagePullPolicy: "IfNotPresent",
					Command: []string{
						"/bin/sh", "-cex", j.Shell,
					},
					WorkingDir: "/build",
					Env: env,
				},
			},
			RestartPolicy: "Never",
		},
	})
	if err != nil {
		return err
	}
	defer clientset.CoreV1().Pods("u-jbarrick").Delete(pod.Name, &metav1.DeleteOptions{})

	watcher, err := clientset.CoreV1().Pods("u-jbarrick").Watch(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("farm/target=%s", j.Name),
	})
	if err != nil {
		return err
	}

	watchCh := watcher.ResultChan()

	for event := range watchCh {
		pod = event.Object.(*corev1.Pod)
		if pod.Status.Phase != "Pending" {
			break
		}
	}

	req := clientset.CoreV1().Pods("u-jbarrick").GetLogs(pod.Name, &corev1.PodLogOptions{})

	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	io.Copy(logger.LogWriter(j), readCloser)

	logger.Log(j, fmt.Sprintf("Checking pod status.\n"))
	pod, err = clientset.CoreV1().Pods("u-jbarrick").Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	exitStatus := pod.Status.ContainerStatuses[0].State.Terminated.ExitCode
	if exitStatus != 0 {
		return errors.New(fmt.Sprintf("Pod exited with error: %d", exitStatus))
	}

	return nil
}
