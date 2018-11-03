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
	"github.com/justinbarrick/farm/pkg/job"
	"github.com/justinbarrick/farm/pkg/logger"
	"io"
	"os"
)

func Run(j job.Job) error {
	ctx := context.TODO()

	d, err := docker.NewEnvClient()
	if err != nil {
		return err
	}

	args := filters.NewArgs()
	args.Add("reference", j.Image)

	images, err := d.ImageList(ctx, types.ImageListOptions{
		Filters: args,
	})
	if err != nil {
		return err
	}

	if len(images) < 1 {
		reader, err := d.ImagePull(ctx, j.Image, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		io.Copy(os.Stdout, reader)
	}

	env := []string{}
	if j.Env != nil {
		for name, value := range *j.Env {
			env = append(env, fmt.Sprintf("%s=%s", name, value))
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	ctr, err := d.ContainerCreate(ctx, &container.Config{
		Image: j.Image,
		Cmd: []string{
			j.Shell,
		},
		Entrypoint: []string{"/bin/sh", "-cex"},
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
		AutoRemove: true,
	}, nil, "")
	if err != nil {
		return err
	}

	if err := d.ContainerStart(ctx, ctr.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	logger.Log(j, fmt.Sprintf("Started container: %s\n", ctr.ID[:8]))

	out, err := d.ContainerLogs(ctx, ctr.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	stdcopy.StdCopy(logger.LogWriter(j), logger.LogWriterError(j), out)

	status, err := d.ContainerWait(ctx, ctr.ID)
	if err != nil {
		return err
	}

	logger.Log(j, fmt.Sprintf("Container exited: %s, status code %d\n", j.Name, status))

	if status != 0 {
		return errors.New(fmt.Sprintf("Container returned status code: %d", status))
	}

	return nil
}
