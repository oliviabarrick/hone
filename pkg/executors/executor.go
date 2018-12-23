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
		if config.Cache.S3 == nil {
			return nil, errors.New("Kubernetes is not currently supported without an S3 configuration.")
		}

		k := &kubernetes.Kubernetes{}

		if config.Kubernetes != nil {
			k = config.Kubernetes
		}

		if k.Cache == nil {
			k.Cache = config.Cache.S3
		}

		orchestrator = k
		logger.Printf("Using Kubernetes for running jobs.\n")
	} else if engine == "local" {
		orchestrator = &local.Local{}
		logger.Printf("Using local for running jobs.\n")
	} else {
		orchestrator = &docker.Docker{}
		logger.Printf("Using Docker for running jobs.\n")
	}

	err := orchestrator.Init()
	if err != nil {
		return nil, err
	}

	return orchestrator, nil
}

func Run(config *types.Config, j *job.Job) error {
	ctx := context.TODO()

	engine, err := ChooseEngine(config, j)
	if err != nil {
		return err
	}

	err = engine.Start(ctx, j)
	if err != nil {
		return err
	}

	finished := make(chan error)

	wait := func() error {
		defer close(finished)

		err = engine.Wait(ctx, j)
		if err != nil {
			finished <- err
			return err
		}

		err = engine.Stop(ctx, j)
		if err != nil {
			finished <- err
		}

		return err
	}

	if j.Service {
		j.Detach <- true

		go wait()

		select {
		case err := <-finished:
			return err
		case <-j.Stop:
			logger.Printf("Got stop!\n")
			return engine.Stop(ctx, j)
		}
	} else {
		return wait()
	}

	return nil
}
