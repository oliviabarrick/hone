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
	"encoding/json"
	"net/http"
	"html/template"
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

	logger.InitLogger(0)

	config, err := config.Unmarshal(honePath)
	if err != nil {
		log.Fatal(err)
	}

	scms, err := scm.InitSCMs(config.SCM, config.Env)
	if err != nil {
		log.Fatal(err)
	}

	report, err := reporting.New(target)
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

	longest, errs := g.LongestTarget(target)
	if len(errs) != 0 {
		done(errs, report, scms, config.Cache.S3)
	}

	logger.InitLogger(longest)

	config.DockerConfig = &docker.DockerConfig{}
	if err := config.DockerConfig.Init(); err != nil {
		log.Fatal(err)
	}

	defer config.DockerConfig.Cleanup()

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
	callback = report.ReportJob(cache.CacheJob(fileCache, callback))

	go http.ListenAndServe("localhost:6060", nil)

	errs = g.ResolveTarget(target, logger.LogJob(callback))
	done(errs, report, scms, config.Cache.S3)
}

func done(errs []error, report reporting.Report, scms []*scm.SCM, cache cache.Cache) {
	report.Success = len(errs) == 0

	file, err := os.Create("report.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.NewEncoder(file).Encode(report)
	if err != nil {
		log.Fatal(err)
	}

	file.Close()

	template, err := template.ParseFiles("index.html")
	if err != nil {
		log.Fatal(err)
	}

	if cache != nil && cache.Enabled() {
		entry, err := cache.Set("report-blobs", "report.json")
		if err != nil {
			log.Fatal(err)
		}

		reportFile, err := os.Create("report.html")
		if err != nil {
			log.Fatal(err)
		}

		template.Execute(reportFile, struct{
			ReportJSON string
		}{
			ReportJSON: fmt.Sprintf("/report-blobs/%s", entry.Hash),
		})

		reportFile.Close()

		entry, err = cache.Set("reports", "report.html")
		if err != nil {
			log.Fatal(err)
		}

		logger.Printf("Report uploaded to: %s/reports/%s", cache.BaseURL(), entry.Hash)
	}

	if len(errs) != 0 {
		if errs[0].Error() == fmt.Sprintf("Target %s not found.", report.Target) {
			logger.Printf("Error: Target %s not found in configuration!", report.Target)
		}

		logger.Errorf("Exiting with failure.")

		if err = scm.BuildErrored(scms); err != nil {
			log.Fatal(err)
		}

		os.Exit(len(errs))
	}

	if err = scm.BuildCompleted(scms); err != nil {
		log.Fatal(err)
	}

	logger.Successf("Build completed successfully!")
}
