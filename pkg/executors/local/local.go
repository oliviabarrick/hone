package local

import (
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/job"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"fmt"
)

func Run(c cache.Cache, j *job.Job) error {
	return Exec(j.GetShell(), j.GetEnv())
}

func ParseEnv(env []string) map[string]string {
	envMap := map[string]string{}

	for _, envVar := range env {
		envSplit := strings.SplitN(envVar, "=", 2)
		envMap[envSplit[0]] = envSplit[1]
	}

	return envMap
}

func Exec(command []string, env map[string]string) error {
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

	stdoutT := io.TeeReader(stdout, os.Stdout)
	stderrT := io.TeeReader(stderr, os.Stderr)

	if err = cmd.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(ioutil.Discard, io.MultiReader(stdoutT, stderrT)); err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return err
	}

	return nil
}
