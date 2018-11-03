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

const Bucket = "justinbarrick-cache"

type S3Cache struct {
	bucket string
	s3 *minio.Client
}

func NewS3Cache(bucket, endpoint, accessKey, secretKey string) (*S3Cache, error) {
	minioClient, err := minio.New(endpoint, accessKey, secretKey, true)
	if err != nil {
		return nil, err
	}

	err = minioClient.MakeBucket(Bucket, "us-east-1")
	if err != nil {
		exists, newErr := minioClient.BucketExists(Bucket)
		if newErr != nil {
			return nil, newErr
		} else if ! exists {
			return nil, err
		}
	}

	logger.Printf("Initialized S3 cache.")

	return &S3Cache{
		s3: minioClient,
	}, nil
}

func (c S3Cache) Name() string {
	return "s3"
}

func (c *S3Cache) Get(entry cache.CacheEntry) error {
	cachePath := filepath.Join("out", entry.Hash)

	err := c.s3.FGetObject(Bucket, cachePath, entry.Filename, minio.GetObjectOptions{})
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

	_, err = c.s3.FPutObject(Bucket, cachePath, filePath, minio.PutObjectOptions{})
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

	object, err := c.s3.GetObject(Bucket, cachePath, minio.GetObjectOptions{})
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

	_, err = c.s3.PutObject(Bucket, cachePath, uploader, -1, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}
