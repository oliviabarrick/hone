package main

import (
	"log"
	"os"
	"github.com/justinbarrick/farm/pkg/config"
	"github.com/justinbarrick/farm/pkg/executors/docker"
	"github.com/justinbarrick/farm/pkg/graph"
	"github.com/justinbarrick/farm/pkg/logger"
	"github.com/justinbarrick/farm/pkg/cache"
)

func main() {
	jobs, err := config.Unmarshal(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	g := graph.NewJobGraph(jobs)
	if err := g.ResolveTarget(os.Args[2], logger.LogJob(cache.CacheJob(func(j config.Job) error {
		return docker.Run(j)
	}))); err != nil {
		log.Fatal(err)
	}
}
