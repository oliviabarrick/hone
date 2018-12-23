package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
	"io"
	"os"
)

type Docker struct {
	docker *docker.Client
	ctr    string
}

func (d *Docker) Init() error {
	if d.docker == nil {
		dockerClient, err := docker.NewEnvClient()
		if err != nil {
			return err
		}

		d.docker = dockerClient
	}

	d.docker.NegotiateAPIVersion(context.TODO())
	return nil
}

func (d *Docker) Pull(ctx context.Context, image string) error {
	args := filters.NewArgs()
	args.Add("reference", image)

	images, err := d.docker.ImageList(ctx, types.ImageListOptions{
		Filters: args,
	})
	if err != nil {
		return err
	}

	if len(images) < 1 {
		reader, err := d.docker.ImagePull(ctx, image, types.ImagePullOptions{})
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
	out, err := d.docker.ContainerLogs(ctx, d.ctr, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	stdcopy.StdCopy(logger.LogWriter(j), logger.LogWriterError(j), out)

	statusCh, errCh := d.docker.ContainerWait(ctx, d.ctr, container.WaitConditionNotRunning)
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
	if statusCode != 0 {
		return errors.New(fmt.Sprintf("Container returned status code: %d", statusCode))
	}

	return nil
}

func (d *Docker) Stop(ctx context.Context, j *job.Job) error {
	return d.docker.ContainerRemove(ctx, d.ctr, types.ContainerRemoveOptions{})
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

	ctr, err := d.docker.ContainerCreate(ctx, &container.Config{
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
	}, nil, "")
	if err != nil {
		return err
	}

	if err := d.docker.ContainerStart(ctx, ctr.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	d.ctr = ctr.ID
	return nil
}
