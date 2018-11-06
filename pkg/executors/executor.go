package executors

import (
	"errors"
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/config/types"
	"github.com/justinbarrick/hone/pkg/executors/docker"
	"github.com/justinbarrick/hone/pkg/executors/kubernetes"
	"github.com/justinbarrick/hone/pkg/executors/local"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
)

func ChooseEngine(config *types.Config, j *job.Job) (func(cache.Cache, *job.Job) error, error) {
	orchestratorCb := docker.Run

	engine := j.GetEngine()
	if engine == "" {
		engine = config.GetEngine()
	}

	if engine == "kubernetes" {
		if config.Cache.S3 == nil {
			return nil, errors.New("Kubernetes is not currently supported without an S3 configuration.")
		}

		k := kubernetes.Kubernetes{}
		if config.Kubernetes != nil {
			k = *config.Kubernetes
		}

		orchestratorCb = k.Run
		logger.Printf("Using Kubernetes for running jobs.\n")
	} else if engine == "local" {
		orchestratorCb = local.Run
		logger.Printf("Using local for running jobs.\n")
	} else {
		logger.Printf("Using Docker for running jobs.\n")
	}

	return orchestratorCb, nil
}
