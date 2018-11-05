package filecache

import (
	"encoding/json"
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/logger"
	"io"
	"os"
	"path/filepath"
)

type FileCache struct {
	CacheDir string `hcl:"cache_dir"`
}

func (c *FileCache) Init() error {
	if c.CacheDir == "" {
		c.CacheDir = ".hone_cache"
	}

	err := os.Mkdir(c.CacheDir, 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.Mkdir(filepath.Join(c.CacheDir, "in"), 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.Mkdir(filepath.Join(c.CacheDir, "out"), 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}

	logger.Printf("Initialized file cache.\n")
	return nil
}

func (c FileCache) Name() string {
	return "file"
}

func (c FileCache) Env() map[string]string {
	return map[string]string{}
}

func (c *FileCache) Copy(src, dst string) error {
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

func (c *FileCache) Get(namespace string, entry cache.CacheEntry) error {
	cacheKey := filepath.Join(c.CacheDir, namespace, entry.Hash)
	err := c.Copy(cacheKey, entry.Filename)
	if err != nil {
		return err
	}
	return nil
}

func (c *FileCache) Set(namespace, filePath string) (cache.CacheEntry, error) {
	cacheKey, err := cache.HashFile(filePath)
	if err != nil {
		return cache.CacheEntry{}, err
	}

	cacheOut := filepath.Join(c.CacheDir, namespace, cacheKey)

	c.Copy(filePath, cacheOut)

	return cache.CacheEntry{
		Filename: filePath,
		Hash:     cacheKey,
	}, nil
}

func (c *FileCache) LoadCacheManifest(namespace, cacheKey string) ([]cache.CacheEntry, error) {
	cachePath := filepath.Join(c.CacheDir, namespace, cacheKey)

	cacheFile, err := os.Open(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}
	defer cacheFile.Close()

	entries := []cache.CacheEntry{}
	err = json.NewDecoder(cacheFile).Decode(&entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (c *FileCache) DumpCacheManifest(namespace, cacheKey string, entries []cache.CacheEntry) error {
	cachePath := filepath.Join(c.CacheDir, namespace, cacheKey)

	cacheFile, err := os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	return json.NewEncoder(cacheFile).Encode(entries)
}
