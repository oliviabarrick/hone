package cache

import (
	"crypto/sha256"
	"fmt"
	config "github.com/justinbarrick/farm/pkg/job"
	"github.com/justinbarrick/farm/pkg/logger"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/bmatcuk/doublestar"
)

type CacheEntry struct {
	Filename string
	Hash     string
	FileMode os.FileMode
}

type Cache interface {
	Name() string
	Get(entry CacheEntry) error
	Set(filePath string) (CacheEntry, error)
	LoadCacheManifest(cacheKey string) ([]CacheEntry, error)
	DumpCacheManifest(cacheKey string, entries []CacheEntry) error
}

func WalkInputs(job config.Job, fn func(string) error) error {
	inputs := []string{}

	if job.Inputs != nil {
		inputs = append(inputs, *job.Inputs...)
	}

	for _, input := range inputs {
		inputFile, err := os.Open(input)
		if err != nil && os.IsNotExist(err) {
			matches, err := doublestar.Glob(input)
			if err != nil {
				return err
			}

			for _, match := range matches {
				inputFile, err := os.Open(match)
				if err != nil {
					continue
				}
				fi, err := inputFile.Stat()
				if err != nil {
					continue
				}

				if fi.IsDir() {
					continue
				}

				err = fn(match)
				if err != nil {
					return err
				}
			}

			continue
		} else if err != nil {
			return err
		}

		fi, err := inputFile.Stat()
		switch {
		case err != nil:
			return err
		case fi.IsDir():
			err = filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					return fn(path)
				}
				return nil
			})
			if err != nil {
				return err
			}
		default:
			err = fn(input)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func HashJob(job config.Job) (string, error) {
	sum := sha256.New()

	sum.Write([]byte(job.Image))
	sum.Write([]byte(job.Shell))

	err := WalkInputs(job, func(path string) error {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		sum.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sum.Sum(nil)), nil
}

func HashFile(filePath string) (string, error) {
	fileSum := sha256.New()

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
  }

	fileSum.Write(data)
	return fmt.Sprintf("%x", fileSum.Sum(nil)), nil
}

func (c *CacheEntry) LoadAttrs() (error) {
	file, err := os.Open(c.Filename)
	if err != nil {
		return err
	}

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	c.FileMode = fi.Mode()
	return nil
}

func (c CacheEntry) SyncAttrs() (error) {
	return os.Chmod(c.Filename, c.FileMode)
}

func CacheJob(c Cache, callback func(config.Job) error) func(config.Job) error {
	return func(job config.Job) error {
		cacheKey, err := HashJob(job)
		if err != nil {
			return err
		}

		cacheManifest, err := c.LoadCacheManifest(cacheKey)
		if err != nil {
			return err
		}

		if cacheManifest != nil {
			for _, entry := range cacheManifest {
				fetch := true

				_, err = os.Open(entry.Filename)
				if err == nil {
					hash, _ := HashFile(entry.Filename)
					if hash == entry.Hash {
						logger.Log(job, fmt.Sprintf("Skipping upto date file %s.", entry.Filename))
						fetch = false
					}
				}

				if fetch {
					err := c.Get(entry)
					if err != nil {
						return err
					}
					err = entry.SyncAttrs()
					if err != nil {
						return err
					}
					logger.Log(job, fmt.Sprintf("Loaded %s from cache (%s).", entry.Filename, c.Name()))
				}
			}

			logger.Log(job, "Job cached.")
			return nil
		}

		err = callback(job)
		if err != nil {
			return err
		}

		entries := []CacheEntry{}
		if job.Outputs != nil {
			for _, output := range *job.Outputs {
				logger.Log(job, fmt.Sprintf("Dumping %s to cache (%s).", output, c.Name()))
				cacheEntry, err := c.Set(output)
				if err != nil {
					return err
				}
				err = cacheEntry.LoadAttrs()
				if err != nil {
					return err
				}
				entries = append(entries, cacheEntry)
			}
		}

		return c.DumpCacheManifest(cacheKey, entries)
	}
}
