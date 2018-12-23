package main

import (
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/config"
	"github.com/justinbarrick/hone/pkg/executors"
	"github.com/justinbarrick/hone/pkg/graph"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/events"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/justinbarrick/hone/pkg/scm"
	"log"
	"os"
	"fmt"
)


func main() {
	honePath := "Honefile"
	target := "all"

	if len(os.Args) == 2 {
		target = os.Args[1]
	} else if len(os.Args) == 3 {
		honePath = os.Args[1]
		target = os.Args[2]
	}

	config, err := config.Unmarshal(honePath)
	if err != nil {
		log.Fatal(err)
	}

	callback := func(j *job.Job) error {
		return executors.Run(config, j)
	}

	callback = events.EventCallback(config.Env, callback)

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

	scms, err := scm.InitSCMs(config.SCM, config.Env)
	if err != nil {
		log.Fatal(err)
	}

	if err = scm.BuildStarted(scms); err != nil {
		log.Fatal(err)
	}

	g, err := graph.NewJobGraph(config.GetJobs())
	if err != nil {
		log.Fatal(err)
	}

	if errs := g.ResolveTarget(target, logger.LogJob(callback)); len(errs) != 0 {
		if errs[0].Error() == fmt.Sprintf("Target %s not found.", target) {
			logger.Printf("Error: Target %s not found in configuration!\n", target)
		}

		logger.Printf("Exiting with failure.\n")

		if err = scm.BuildErrored(scms); err != nil {
			log.Fatal(err)
		}

		os.Exit(len(errs))
	}

	if err = scm.BuildCompleted(scms); err != nil {
		log.Fatal(err)
	}
}
