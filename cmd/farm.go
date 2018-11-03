package main

import (
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/cache/s3"
	"github.com/justinbarrick/farm/pkg/cache/file"
	"github.com/justinbarrick/farm/pkg/config"
	"github.com/justinbarrick/farm/pkg/executors/docker"
	"github.com/justinbarrick/farm/pkg/graph"
	"github.com/justinbarrick/farm/pkg/logger"
	"log"
	"os"
)

func main() {
	jobs, err := config.Unmarshal(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	callback := func(j config.Job) error {
		return docker.Run(j)
	}

	c, err := filecache.NewFileCache(".farm_cache")
	if err != nil {
		log.Fatal(err)
	}
	callback = cache.CacheJob(c, callback)

	bucket := os.Getenv("S3_BUCKET")
	endpoint := os.Getenv("S3_URL")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	if endpoint != "" && accessKey != "" && secretKey != "" && bucket != "" {
		c, err := s3cache.NewS3Cache(bucket, endpoint, accessKey, secretKey)
		if err != nil {
			log.Fatal(err)
		}
		callback = cache.CacheJob(c, callback)
	}

	g := graph.NewJobGraph(jobs)
	if err := g.ResolveTarget(os.Args[2], logger.LogJob(callback)); err != nil {
		log.Fatal(err)
	}
}
