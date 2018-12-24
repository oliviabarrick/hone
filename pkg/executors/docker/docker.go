package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
	"io"
	"os"
	"time"
)

type DockerConfig struct {
	docker *docker.Client
	network string
}

func (dc *DockerConfig) Init() error {
	dockerClient, err := docker.NewEnvClient()
	if err != nil {
		return err
	}

	dc.docker = dockerClient
	dc.docker.NegotiateAPIVersion(context.TODO())

	return dc.CreateNetwork()
}

func (dc *DockerConfig) Cleanup() error {
	return dc.DeleteNetwork()
}

func (dc *DockerConfig) CreateNetwork() error {
	network, err := dc.docker.NetworkCreate(context.TODO(), "hone", types.NetworkCreate{})
	if err != nil {
		return err
	}

	dc.network = network.ID
	return nil
}

func (dc *DockerConfig) DeleteNetwork() error {
	return dc.docker.NetworkRemove(context.TODO(), dc.network)
}

type Docker struct {
	DockerConfig *DockerConfig
	ctr    string
}

func (d *Docker) Init() error {
	return nil
}

func (d *Docker) Pull(ctx context.Context, image string) error {
	args := filters.NewArgs()
	args.Add("reference", image)

	images, err := d.DockerConfig.docker.ImageList(ctx, types.ImageListOptions{
		Filters: args,
	})
	if err != nil {
		return err
	}

	if len(images) < 1 {
		reader, err := d.DockerConfig.docker.ImagePull(ctx, image, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Docker) Wait(ctx context.Context, j *job.Job) error {
	logger.Log(j, fmt.Sprintf("Started container: %s\n", d.ctr[:8]))
	out, err := d.DockerConfig.docker.ContainerLogs(ctx, d.ctr, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	stdcopy.StdCopy(logger.LogWriter(j), logger.LogWriterError(j), out)

	statusCh, errCh := d.DockerConfig.docker.ContainerWait(ctx, d.ctr, container.WaitConditionNotRunning)
	statusCode := int64(0)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
	}

	logger.Log(j, fmt.Sprintf("Container exited: %s, status code %d\n", j.GetName(), statusCode))
	if statusCode != 128 && statusCode != 0 {
		return errors.New(fmt.Sprintf("Container returned status code: %d", statusCode))
	} else if ! j.Service && statusCode == 128 {
		return errors.New(fmt.Sprintf("Container returned status code: %d", statusCode))
	}

	return nil
}

func (d *Docker) Stop(ctx context.Context, j *job.Job) error {
	timeout := 5 * time.Second
	d.DockerConfig.docker.ContainerStop(ctx, d.ctr, &timeout)
	return d.DockerConfig.docker.ContainerRemove(ctx, d.ctr, types.ContainerRemoveOptions{})
}

func (d *Docker) Start(ctx context.Context, j *job.Job) error {
	err := d.Pull(ctx, j.GetImage())
	if err != nil {
		return err
	}

	env := []string{}
	for name, value := range j.GetEnv() {
		env = append(env, fmt.Sprintf("%s=%s", name, value))
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	ctr, err := d.DockerConfig.docker.ContainerCreate(ctx, &container.Config{
		Hostname:   j.GetName(),
		Image:      j.GetImage(),
		Entrypoint: j.GetShell(),
		Env:        env,
		WorkingDir: "/build",
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: cwd,
				Target: "/build",
			},
		},
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"hone": &network.EndpointSettings{
				Aliases: []string{j.GetName()},
				NetworkID: d.DockerConfig.network,
			},
		},
	}, "")
	if err != nil {
		return err
	}

	if err := d.DockerConfig.docker.ContainerStart(ctx, ctr.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	d.ctr = ctr.ID
	return nil
}
