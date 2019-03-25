package s3cache

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"github.com/hashicorp/go-rootcerts"
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/minio/minio-go"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"fmt"
	"path/filepath"
)

type S3Cache struct {
	Bucket    string `hcl:"bucket"`
	Endpoint  string `hcl:"endpoint"`
	AccessKey string `hcl:"access_key"`
	SecretKey string `hcl:"secret_key"`
	Disabled  bool   `hcl:"disabled"`
	s3        *minio.Client
}

func (c *S3Cache) Init() error {
	minioClient, err := minio.New(c.Endpoint, c.AccessKey, c.SecretKey, true)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{}
	if os.Getenv("CA_FILE") != "" {
		err := rootcerts.ConfigureTLS(tlsConfig, &rootcerts.Config{
			CAFile: os.Getenv("CA_FILE"),
		})
		if err != nil {
			return err
		}
	}

	minioClient.SetCustomTransport(&http.Transport{
		TLSClientConfig: tlsConfig,
	})

	err = minioClient.MakeBucket(c.Bucket, "us-east-1")
	if err != nil {
		exists, newErr := minioClient.BucketExists(c.Bucket)
		if newErr != nil {
			return newErr
		} else if !exists {
			return err
		}
	}

	err = minioClient.SetBucketPolicy(c.Bucket, fmt.Sprintf(`{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"AddPerm",
      "Effect":"Allow",
      "Principal": "*",
      "Action":["s3:GetObject"],
      "Resource":[
        "arn:aws:s3:::%s/logs/*",
        "arn:aws:s3:::%s/reports/*",
        "arn:aws:s3:::%s/report-blobs/*"
      ]
    }
  ]
}`, c.Bucket, c.Bucket, c.Bucket))
	if err != nil && err.Error() != "200 OK" {
		return err
	}

	logger.Printf("Initialized S3 cache.")
	c.s3 = minioClient
	return nil
}

func (c S3Cache) Env() map[string]string {
	return map[string]string{
		"S3_BUCKET":     c.Bucket,
		"S3_ENDPOINT":   c.Endpoint,
		"S3_ACCESS_KEY": c.AccessKey,
		"S3_SECRET_KEY": c.SecretKey,
	}
}

func (c S3Cache) Name() string {
	return "s3"
}

func (c *S3Cache) Get(namespace string, entry cache.CacheEntry) error {
	cachePath := filepath.Join(namespace, entry.Hash)

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

func (c *S3Cache) Set(namespace, filePath string) (cache.CacheEntry, error) {
	cacheKey, err := cache.HashFile(filePath)
	if err != nil {
		return cache.CacheEntry{}, err
	}

	cachePath := filepath.Join(namespace, cacheKey)

	_, err = c.s3.FPutObject(c.Bucket, cachePath, filePath, minio.PutObjectOptions{})
	if err != nil {
		return cache.CacheEntry{}, err
	}

	return cache.CacheEntry{
		Filename: filePath,
		Hash:     cacheKey,
	}, nil
}

func (c *S3Cache) LoadCacheManifest(namespace, cacheKey string) ([]cache.CacheEntry, error) {
	cachePath := filepath.Join(namespace, cacheKey)

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

func (c *S3Cache) DumpCacheManifest(namespace, cacheKey string, entries []cache.CacheEntry) error {
	cachePath := filepath.Join(namespace, cacheKey)

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

func (c *S3Cache) Enabled() bool {
	if c == nil {
		return false
	}

	return ! c.Disabled
}

func (c *S3Cache) BaseURL() string {
	return fmt.Sprintf("https://%s.%s", c.Bucket, c.Endpoint)
}

type S3Writer struct {
	writer io.WriteCloser
	done   chan error
}

func (w *S3Writer) Init(s3 *S3Cache, namespace, filename string) string {
	reader, writer := io.Pipe()

	path := filepath.Join(namespace, filename)
	url := fmt.Sprintf("%s/%s", s3.BaseURL(), path)

	w.done = make(chan error)

	go func() {
		var err error

		if s3.s3 == nil {
			_, err = ioutil.ReadAll(reader)
		} else {
			_, err = s3.s3.PutObject(s3.Bucket, path, reader, -1, minio.PutObjectOptions{
				ContentType: mime.TypeByExtension(filepath.Ext(filename)),
			})
		}

		w.done <- err
	}()

	w.writer = writer
	return url
}

func (w *S3Writer) Write(bytes []byte) (int, error) {
	return w.writer.Write(bytes)
}

func (w *S3Writer) Close() error {
	w.writer.Close()
	return <-w.done
}

func (c *S3Cache) Writer(namespace string, filename string) (io.WriteCloser, string, error) {
	writer := &S3Writer{}
	url := writer.Init(c, namespace, filename)
	return writer, url, nil
}
