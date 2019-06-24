package local

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
)

func ParseEnv(env []string) map[string]string {
	envMap := map[string]string{}

	for _, envVar := range env {
		envSplit := strings.SplitN(envVar, "=", 2)
		envMap[envSplit[0]] = envSplit[1]
	}

	return envMap
}

type Local struct {
	stdout io.Reader
	stderr io.Reader
	cmd    *exec.Cmd
}

func (l *Local) Init() error {
	return nil
}

func (l *Local) Start(ctx context.Context, j *job.Job) error {
	return l.Exec(j.GetShell(), j.GetEnv(), j)
}

func (l *Local) Wait(ctx context.Context, j *job.Job) error {
	return l.WaitCmd()
}

func (l *Local) Stop(ctx context.Context, j *job.Job) error {
	return nil
}

func (l *Local) Exec(command []string, env map[string]string, j *job.Job) error {
	cmd := exec.Command(command[0], command[1:]...)

	envList := []string{}
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envList

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if j == nil {
		l.stdout = io.TeeReader(stdout, os.Stdout)
		l.stderr = io.TeeReader(stderr, os.Stderr)
	} else {
		l.stdout = io.TeeReader(stdout, logger.LogWriter(j))
		l.stderr = io.TeeReader(stderr, logger.LogWriterError(j))
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	l.cmd = cmd
	return nil
}

func (l *Local) WaitCmd() error {
	if _, err := io.Copy(ioutil.Discard, io.MultiReader(l.stdout, l.stderr)); err != nil {
		return err
	}

	if err := l.cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func Exec(command []string, env map[string]string) error {
	l := Local{}

	if err := l.Init(); err != nil {
		return err
	}

	if err := l.Exec(command, env, nil); err != nil {
		return err
	}

	if err := l.WaitCmd(); err != nil {
		return err
	}

	return nil
}
