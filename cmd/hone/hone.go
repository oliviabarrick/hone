package main


import (
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/config"
	"github.com/justinbarrick/hone/pkg/executors"
	"github.com/justinbarrick/hone/pkg/executors/docker"
	"github.com/justinbarrick/hone/pkg/graph"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/events"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/justinbarrick/hone/pkg/scm"
	"github.com/justinbarrick/hone/pkg/reporting"
	_ "net/http/pprof"
	"net/http"
	"log"
	"os"
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

	logger.InitLogger(0)

	config, err := config.Unmarshal(honePath)
	if err != nil {
		log.Fatal(err)
	}

	scms, err := scm.InitSCMs(config.SCM, config.Env)
	if err != nil {
		log.Fatal(err)
	}

	report, err := reporting.New(target, scms, config.Cache.S3)
	if err != nil {
		log.Fatal(err)
	}

	if err = scm.BuildStarted(scms); err != nil {
		report.Exit(err)
	}

	g, err := graph.NewJobGraph(config.GetJobs())
	if err != nil {
		report.Exit(err)
	}

	longest, errs := g.LongestTarget(target)
	if len(errs) != 0 {
		report.Exit(errs...)
	}

	logger.InitLogger(longest)

	config.DockerConfig = &docker.DockerConfig{}
	if err := config.DockerConfig.Init(); err != nil {
		report.Exit(err)
	}

	defer config.DockerConfig.Cleanup()

	callback := func(j *job.Job) error {
		return executors.Run(config, j)
	}

	callback = events.EventCallback(config.Env, callback)

	if config.Cache.S3 != nil && !config.Cache.S3.Disabled {
		if err = config.Cache.S3.Init(); err != nil {
			report.Exit(err)
		}
		callback = cache.CacheJob(config.Cache.S3, callback)
	}

	fileCache := config.Cache.File
	if err = fileCache.Init(); err != nil {
		report.Exit(err)
	}
	callback = report.ReportJob(cache.CacheJob(fileCache, callback))

	go http.ListenAndServe("localhost:6060", nil)

	errs = g.ResolveTarget(target, logger.LogJob(callback))
	report.Exit(errs...)
}
