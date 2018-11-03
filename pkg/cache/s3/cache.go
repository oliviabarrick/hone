package s3cache

import (
	"bytes"
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/logger"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"github.com/minio/minio-go"
)

type S3Cache struct {
	Bucket string `hcl:"bucket"`
	Endpoint string `hcl:"endpoint"`
	AccessKey string `hcl:"access_key"`
	SecretKey string `hcl:"secret_key"`
	s3 *minio.Client
}

func (c *S3Cache) Init() error {
	minioClient, err := minio.New(c.Endpoint, c.AccessKey, c.SecretKey, true)
	if err != nil {
		return err
	}

	err = minioClient.MakeBucket(c.Bucket, "us-east-1")
	if err != nil {
		exists, newErr := minioClient.BucketExists(c.Bucket)
		if newErr != nil {
			return newErr
		} else if ! exists {
			return err
		}
	}

	logger.Printf("Initialized S3 cache.")
	c.s3 = minioClient
	return nil
}

func (c S3Cache) Name() string {
	return "s3"
}

func (c *S3Cache) Get(entry cache.CacheEntry) error {
	cachePath := filepath.Join("out", entry.Hash)

	err := c.s3.FGetObject(c.Bucket, cachePath, entry.Filename, minio.GetObjectOptions{})
	if err != nil {
		if err.Error() != "The specified key does not exist." {
			return err
		} else {
			return nil
		}
	}

	return nil
}

func (c *S3Cache) Set(filePath string) (cache.CacheEntry, error) {
	cacheKey, err := cache.HashFile(filePath)
	if err != nil {
		return cache.CacheEntry{}, err
	}

	cachePath := filepath.Join("out", cacheKey)

	_, err = c.s3.FPutObject(c.Bucket, cachePath, filePath, minio.PutObjectOptions{})
	if err != nil {
		return cache.CacheEntry{}, err
	}

	return cache.CacheEntry{
		Filename: filePath,
		Hash:     cacheKey,
	}, nil
}

func (c *S3Cache) LoadCacheManifest(cacheKey string) ([]cache.CacheEntry, error) {
	cachePath := filepath.Join("in", cacheKey)

	object, err := c.s3.GetObject(c.Bucket, cachePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(object)
	if err != nil {
		if err.Error() != "The specified key does not exist." {
			return nil, err
		} else {
			return nil, nil
		}
	}

	entries := []cache.CacheEntry{}

	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (c *S3Cache) DumpCacheManifest(cacheKey string, entries []cache.CacheEntry) error {
	cachePath := filepath.Join("in", cacheKey)

	encoded, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	uploader := bytes.NewBuffer(encoded)

	_, err = c.s3.PutObject(c.Bucket, cachePath, uploader, -1, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}