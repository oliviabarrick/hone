package main

import (
	"encoding/json"
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/cache/s3"
	"github.com/justinbarrick/farm/pkg/executors/local"
	"github.com/justinbarrick/farm/pkg/logger"
	"log"
	"os"
)

func main() {
	s3 := s3cache.S3Cache{
		Bucket:    os.Getenv("S3_BUCKET"),
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
	}

	if err := s3.Init(); err != nil {
		log.Fatal(err)
	}

	cacheManifest, err := s3.LoadCacheManifest("srcs_manifests", os.Getenv("CACHE_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range cacheManifest {
		err := s3.Get("srcs", entry)
		if err != nil {
			log.Fatal(err)
		}
		err = entry.SyncAttrs()
		if err != nil {
			log.Fatal(err)
		}
		logger.Printf("Loaded %s from cache (%s).\n", entry.Filename, s3.Name())
	}

	outputs := []string{}
	err = json.Unmarshal([]byte(os.Getenv("OUTPUTS")), &outputs)
	if err != nil {
		log.Fatal(err)
	}

	os.Unsetenv("S3_BUCKET")
	os.Unsetenv("S3_ENDPOINT")
	os.Unsetenv("S3_ACCESS_KEY")
	os.Unsetenv("S3_SECRET_KEY")
	os.Unsetenv("CACHE_KEY")
	os.Unsetenv("OUTPUTS")

	if err = local.Exec(os.Args[1:], local.ParseEnv(os.Environ())); err != nil {
		log.Fatal(err)
	}

	if _, err = cache.DumpOutputs(os.Getenv("CACHE_KEY"), &s3, outputs); err != nil {
		log.Fatal(err)
	}
	logger.Printf("Dumped outputs to cache (%s).\n", s3.Name())
}
