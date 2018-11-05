package storage

import (
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/job"
)

func UploadInputs(c cache.Cache, j *job.Job) (string, error) {
	entries := []cache.CacheEntry{}

	err := cache.WalkInputs(j, func(filepath string) error {
		cacheEntry, err := c.Set("srcs", filepath)
		if err != nil {
			return err
		}
		err = cacheEntry.LoadAttrs()
		if err != nil {
			return err
		}
		entries = append(entries, cacheEntry)
		return nil
	})

	if err != nil {
		return "", err
	}

	cacheKey, err := cache.HashJob(j)
	if err != nil {
		return "", err
	}

	err = c.DumpCacheManifest("srcs_manifests", cacheKey, entries)
	if err != nil {
		return "", err
	}

	return cacheKey, nil
}
