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
	"github.com/justinbarrick/hone/pkg/graph/node"
)

type Config struct {
	Env          map[string]string
	Secrets      map[string]string
	SCM          []*scm.SCM
	Jobs         []*job.Job
	Cache        CacheConfig
	Kubernetes   *kubernetes.Kubernetes
	DockerConfig *docker.DockerConfig
	Engine       *string
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

func (c Config) GetNodes() []node.Node {
	nodes := []node.Node{}

	for _, job := range c.Jobs {
		nodes = append(nodes, job)
	}

	return nodes
}
