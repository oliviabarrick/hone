package cache

import (
	"crypto/sha256"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/cnf/structhash"
	config "github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

type CacheEntry struct {
	Filename string
	Hash     string
	FileMode os.FileMode
}

type Cache interface {
	Name() string
	Env() map[string]string
	Get(namespace string, entry CacheEntry) error
	Set(namespace, filePath string) (CacheEntry, error)
	LoadCacheManifest(namespace, cacheKey string) ([]CacheEntry, error)
	DumpCacheManifest(namespace, cacheKey string, entries []CacheEntry) error
	Enabled() bool
	BaseURL() string
	Writer(string, string) (io.WriteCloser, string, error)
}

func WalkInputs(job *config.Job, fn func(string) error) error {
	for _, input := range job.GetInputs() {
		inputFile, err := os.Open(input)
		if err != nil && os.IsNotExist(err) {
			matches, err := doublestar.Glob(input)
			if err != nil {
				return err
			}

			sort.Strings(matches)

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

func HashJob(job *config.Job) (string, error) {
	sum := sha256.New()

	sum.Write(structhash.Sha1(job, 1))

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

func (c *CacheEntry) LoadAttrs() error {
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

func (c CacheEntry) SyncAttrs() error {
	return os.Chmod(c.Filename, c.FileMode)
}

func CacheJob(c Cache, callback func(*config.Job) error) func(*config.Job) error {
	return func(job *config.Job) error {
		if job.IsService() {
			return callback(job)
		}

		cacheKey, err := HashJob(job)
		if err != nil {
			return err
		}

		job.Hash = cacheKey

		cached, err := LoadCache(c, cacheKey, job)
		if err != nil {
			return err
		}

		if !cached {
			err = callback(job)
			if err != nil {
				return err
			}
		} else {
			logger.LogDebug(job, "Job cached.")
			job.Cached = true
		}

		if len(job.GetOutputs()) == 0 && len(job.GetInputs()) == 0 {
			return nil
		}

		logger.LogDebug(job, fmt.Sprintf("Dumping to cache (%s).", c.Name()))
		entries, err := DumpOutputs(cacheKey, c, job.GetOutputs())
		if err != nil {
			return err
		}

		if job.OutputHashes == nil {
			job.OutputHashes = map[string]string{}
		}

		for _, entry := range entries {
			job.OutputHashes[entry.Filename] = entry.Hash
		}

		return nil
	}
}

func LoadCache(c Cache, cacheKey string, job *config.Job) (bool, error) {
	cacheManifest, err := c.LoadCacheManifest("in", cacheKey)
	if err != nil {
		return false, err
	}

	if cacheManifest != nil {
		for _, entry := range cacheManifest {
			fetch := true

			_, err = os.Open(entry.Filename)
			if err == nil {
				hash, _ := HashFile(entry.Filename)
				if hash == entry.Hash {
					logger.LogDebug(job, fmt.Sprintf("Skipping upto date file %s.", entry.Filename))
					fetch = false
				}
			}

			if fetch {
				err := c.Get("out", entry)
				if err != nil {
					return false, err
				}
				err = entry.SyncAttrs()
				if err != nil {
					return false, err
				}
				logger.LogDebug(job, fmt.Sprintf("Loaded %s from cache (%s).", entry.Filename, c.Name()))
			}
		}

		return true, nil
	}

	return false, nil
}

func DumpOutputs(cacheKey string, c Cache, outputs []string) ([]CacheEntry, error) {
	entries := []CacheEntry{}

	for _, output := range outputs {
		cacheEntry, err := c.Set("out", output)
		if err != nil {
			return nil, err
		}
		err = cacheEntry.LoadAttrs()
		if err != nil {
			return nil, err
		}
		entries = append(entries, cacheEntry)
	}

	err := c.DumpCacheManifest("in", cacheKey, entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}
