package executors

import (
	"context"
	"errors"

	"github.com/justinbarrick/hone/pkg/config/types"
	"github.com/justinbarrick/hone/pkg/executors/docker"
	"github.com/justinbarrick/hone/pkg/executors/kubernetes"
	"github.com/justinbarrick/hone/pkg/executors/local"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
)

type Engine interface {
	Init() error
	Start(context.Context, *job.Job) error
	Wait(context.Context, *job.Job) error
	Stop(context.Context, *job.Job) error
}

func ChooseEngine(config *types.Config, j *job.Job) (Engine, error) {
	engine := j.GetEngine()
	if engine == "" {
		engine = config.GetEngine()
	}

	var orchestrator Engine

	if engine == "kubernetes" {
		if config.Cache.S3 == nil || config.Cache.S3.Disabled {
			return nil, errors.New("Kubernetes is not currently supported without an S3 configuration.")
		}

		k := kubernetes.Kubernetes{}

		if config.Kubernetes != nil {
			k = *config.Kubernetes
		}

		if k.Cache == nil {
			k.Cache = config.Cache.S3
		}

		orchestrator = &k
		logger.Log(j, "Using Kubernetes for running job.")
	} else if engine == "local" {
		orchestrator = &local.Local{}
		logger.Log(j, "Using local for running job.")
	} else {
		orchestrator = &docker.Docker{
			DockerConfig: config.DockerConfig,
		}

		logger.Log(j, "Using Docker for running job.")
	}

	err := orchestrator.Init()
	if err != nil {
		return nil, err
	}

	return orchestrator, nil
}

func Run(config *types.Config, j *job.Job) error {
	ctx := context.TODO()
	finished := make(chan error)

	engine, err := ChooseEngine(config, j)
	if err != nil {
		return err
	}

	err = engine.Start(ctx, j)
	defer engine.Stop(ctx, j)
	if err != nil {
		return err
	}

	if j.IsService() {
		j.Detach <- true
	}

	go func() {
		finished <- engine.Wait(ctx, j)
	}()

	select {
	case err := <-finished:
		return err
	case <-j.Stop:
	}

	return nil
}
