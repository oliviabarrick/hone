package main

import (
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/config"
	"github.com/justinbarrick/farm/pkg/executors"
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

	callback := func(j *job.Job) error {
		orchestratorCb, err := executors.ChooseEngine(config, j)
		if err != nil {
			return err
		}

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
