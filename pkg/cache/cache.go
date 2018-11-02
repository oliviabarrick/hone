package cache

import (
	"fmt"
	"os"
	"log"
	"io"
	"path/filepath"
	"io/ioutil"
	"encoding/json"
	"crypto/sha256"
	"github.com/justinbarrick/farm/pkg/config"
)

type CacheEntry struct {
	Filename string
	Hash string
}

type Cache struct {
	CacheDir string
}

func NewCache(cacheDir string) (Cache, error) {
	cache := Cache{
		CacheDir: cacheDir,
	}

	err := os.Mkdir(cacheDir, 0777)
	if err != nil && ! os.IsExist(err) {
		return Cache{}, err
	}

	err = os.Mkdir(filepath.Join(cacheDir, "in"), 0777)
	if err != nil && ! os.IsExist(err) {
		return Cache{}, err
	}

	err = os.Mkdir(filepath.Join(cacheDir, "out"), 0777)
	if err != nil && ! os.IsExist(err) {
		return Cache{}, err
	}


	return cache, nil
}

func (c *Cache) Copy(src, dst string) error {
  from, err := os.Open(src)
  if err != nil {
		return err
  }
  defer from.Close()

  to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
  if err != nil {
    return err
  }
  defer to.Close()

  _, err = io.Copy(to, from)
  if err != nil {
    return err
  }

	return nil
}

func (c *Cache) Get(entry CacheEntry) error {
	cacheKey := filepath.Join(c.CacheDir, "out", entry.Hash)
	return c.Copy(cacheKey, entry.Filename)
}

func (c *Cache) Set(filePath string) (CacheEntry, error) {
	fileSum := sha256.New()

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return CacheEntry{}, err
	}

	fileSum.Write(data)

	cacheKey := fmt.Sprintf("%x", fileSum.Sum(nil))
	cacheOut := filepath.Join(c.CacheDir, "out", cacheKey)

	c.Copy(filePath, cacheOut)

	return CacheEntry{
		Filename: filePath,
		Hash: cacheKey,
	}, nil
}

func (c *Cache) LoadCacheManifest(cacheKey string) ([]CacheEntry, error) {
	cachePath := filepath.Join(c.CacheDir, "in", cacheKey)

	cacheFile, err := os.Open(cachePath)
  if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
  }
  defer cacheFile.Close()

	entries := []CacheEntry{}
	err = json.NewDecoder(cacheFile).Decode(&entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (c *Cache) DumpCacheManifest(cacheKey string, entries []CacheEntry) error {
	cachePath := filepath.Join(c.CacheDir, "in", cacheKey)

  cacheFile, err := os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE, 0666)
  if err != nil {
		return err
  }
  defer cacheFile.Close()

	return json.NewEncoder(cacheFile).Encode(entries)
}

func (c *Cache) WalkInputs(job config.Job, fn func(string) error) error {
	inputs := []string{}
	inputs = append(inputs, job.Inputs...)

	for _, input := range inputs {
		inputFile, err := os.Open(input)
		if err != nil && os.IsNotExist(err) {
			matches, err := filepath.Glob(input)
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
				if ! info.IsDir() {
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

func (c *Cache) HashJob(job config.Job) (string, error) {
	sum := sha256.New()

	sum.Write([]byte(job.Image))
	sum.Write([]byte(job.Shell))

	err := c.WalkInputs(job, func(path string) error {
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

func CacheJob(callback func (config.Job) error) func (config.Job) error {
	return func (job config.Job) error {
		c, err := NewCache(".farm_cache")
		if err != nil {
			return err
		}

		cacheKey, err := c.HashJob(job)
		if err != nil {
			return err
		}

		cacheManifest, err := c.LoadCacheManifest(cacheKey)
		if err != nil {
			return err
		}

		if cacheManifest != nil {
			for _, entry := range cacheManifest {
				err := c.Get(entry)
				if err != nil {
					return err
				}
			}

			log.Printf("Loaded job \"%s\" from cache.\n", job.Name)
			return nil
		}

		err = callback(job)
		if err != nil {
			return err
		}

		entries := []CacheEntry{}
		for _, output := range job.Outputs {
			cacheEntry, err := c.Set(output)
			if err != nil {
				return err
			}
			entries = append(entries, cacheEntry)
		}

		return c.DumpCacheManifest(cacheKey, entries)
	}
}
