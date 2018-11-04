package main

import (
	"encoding/json"
	"github.com/justinbarrick/farm/pkg/logger"
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/cache/s3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
		logger.Printf("Loaded %s from cache (%s).\n", entry.Filename, s3.Name())
	}

	outputs := []string{}
	err = json.Unmarshal([]byte(os.Getenv("OUTPUTS")), &outputs)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	
	stdoutT := io.TeeReader(stdout, os.Stdout)
	stderrT := io.TeeReader(stderr, os.Stderr)

	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	if _, err := io.Copy(ioutil.Discard, io.MultiReader(stdoutT, stderrT)); err != nil {
		log.Fatal(err)
	}

	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}


	if _, err = cache.DumpOutputs(os.Getenv("CACHE_KEY"), &s3, outputs); err != nil {
		log.Fatal(err)
	}
	logger.Printf("Dumped outputs to cache (%s).\n", s3.Name())
}
