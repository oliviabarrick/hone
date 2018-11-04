package main

import (
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/config"
	"github.com/justinbarrick/farm/pkg/executors/docker"
	"github.com/justinbarrick/farm/pkg/executors/kubernetes"
	"github.com/justinbarrick/farm/pkg/graph"
	"github.com/justinbarrick/farm/pkg/job"
	"github.com/justinbarrick/farm/pkg/logger"
	"log"
	"os"
)

func main() {
	farmPath := ".farm.hcl"
	target := "all"

	if len(os.Args) == 2 {
		target = os.Args[1]
	} else if len(os.Args) == 3 {
		farmPath = os.Args[1]
		target = os.Args[2]
	}

	config, err := config.Unmarshal(farmPath)
	if err != nil {
		log.Fatal(err)
	}

	orchestratorCb := docker.Run
	if config.Engine != nil && *config.Engine == "kubernetes" {
		if config.Cache.S3 == nil {
			log.Fatal("Kubernetes is not currently supported without an S3 configuration.")
		}

		k := kubernetes.Kubernetes{}
		if config.Kubernetes == nil {
			k = *config.Kubernetes
		}

		orchestratorCb = k.Run
		logger.Printf("Using Kubernetes for running jobs.\n")
	} else {
		logger.Printf("Using Docker for running jobs.\n")
	}

	callback := func(j *job.Job) error {
		return orchestratorCb(config.Cache.S3, j)
	}

	if config.Cache.S3 != nil && !config.Cache.S3.Disabled {
		if err = config.Cache.S3.Init(); err != nil {
			log.Fatal(err)
		}
		callback = cache.CacheJob(config.Cache.S3, callback)
	}

	fileCache := config.Cache.File
	if err = fileCache.Init(); err != nil {
		log.Fatal(err)
	}
	callback = cache.CacheJob(fileCache, callback)

	g := graph.NewJobGraph(config.Jobs)
	if errs := g.ResolveTarget(target, logger.LogJob(callback)); len(errs) != 0 {
		logger.Printf("Exiting with failure.\n")
		os.Exit(len(errs))
	}
}
