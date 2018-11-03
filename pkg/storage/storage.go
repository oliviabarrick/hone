package storage

import (
	"os"
	"os/user"
	"path/filepath"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/justinbarrick/farm/pkg/job"
)

func Run(j job.Job) error {
	kubeconfig := os.Getenv("KUBECONFIG")
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

	_, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return nil
}
