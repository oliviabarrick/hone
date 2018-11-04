package main

import (
	"github.com/justinbarrick/farm/pkg/cache/s3"
	"log"
	"os"
)

func main() {
	s3 := s3cache.S3Cache{
		Bucket: os.Getenv("S3_BUCKET"),
		Endpoint: os.Getenv("S3_ENDPOINT"),
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
		log.Printf("Loaded %s from cache (%s).\n", entry.Filename, s3.Name())
	}
}
