package types

import (
	"errors"
	"fmt"
	"github.com/justinbarrick/hone/pkg/cache/file"
	"github.com/justinbarrick/hone/pkg/cache/s3"
	"github.com/justinbarrick/hone/pkg/executors/kubernetes"
	"github.com/justinbarrick/hone/pkg/executors/docker"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/scm"
)

type Config struct {
	Env        map[string]interface{}
	SCM        []*scm.SCM             `hcl:"report,block"`
	Jobs       []*job.Job             `hcl:"job,block"`
	Services   []*job.Job             `hcl:"service,block"`
	Cache      *CacheConfig           `hcl:"cache,block"`
	Kubernetes *kubernetes.Kubernetes `hcl:"kubernetes,block"`
	DockerConfig *docker.DockerConfig
	Engine     *string                `hcl:"engine"`
}

type CacheConfig struct {
	S3   *s3cache.S3Cache     `hcl:"s3,block"`
	File *filecache.FileCache `hcl:"file,block"`
}

func (c Config) Validate() error {
	for _, job := range c.Jobs {
		if err := job.Validate(c.GetEngine()); err != nil {
			return errors.New(fmt.Sprintf("Error validating job %s: %s", job.GetName(), err))
		}
	}

	for _, service := range c.Services {
		service.Service = true
		if err := service.Validate(c.GetEngine()); err != nil {
			return errors.New(fmt.Sprintf("Error validating service %s: %s", service.GetName(), err))
		}
	}

	return nil
}

func (c Config) RenderTemplates(templates []*job.Job) error {
	templateMap := map[string]*job.Job{}

	for _, template := range templates {
		templateMap[template.Name] = template
	}

	for _, job := range c.Jobs {
		template := "default"
		if job.Template != nil {
			template = *job.Template
		}

		if templateMap[template] != nil {
			job.Default(*templateMap[template])
		}
	}

	return nil
}

func (c Config) GetEngine() string {
	if c.Engine != nil {
		return *c.Engine
	}
	return ""
}

func (c Config) GetJobs() []*job.Job {
	jobs := []*job.Job{}

	if c.Services != nil {
		jobs = append(jobs, c.Services...)
	}

	if c.Jobs != nil {
		jobs = append(jobs, c.Jobs...)
	}

	return jobs
}
