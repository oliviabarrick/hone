package main

import (
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/config"
	"github.com/justinbarrick/farm/pkg/job"
	"github.com/justinbarrick/farm/pkg/executors/docker"
	"github.com/justinbarrick/farm/pkg/executors/kubernetes"
	"github.com/justinbarrick/farm/pkg/graph"
	"github.com/justinbarrick/farm/pkg/logger"
	"log"
	"os"
)

func main() {
	config, err := config.Unmarshal(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	orchestratorCb := docker.Run
	if config.Engine != nil && *config.Engine == "kubernetes" {
		orchestratorCb = kubernetes.Run
	}

	callback := func(j *job.Job) error {
		return orchestratorCb(j)
	}

	if config.Cache.S3 != nil && ! config.Cache.S3.Disabled {
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
	if errs := g.ResolveTarget(os.Args[2], logger.LogJob(callback)); len(errs) != 0 {
		log.Println("Exiting with failure.")
		os.Exit(len(errs))
	}
}
