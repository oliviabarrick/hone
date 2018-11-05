package local

import (
	"github.com/justinbarrick/farm/pkg/cache"
	"github.com/justinbarrick/farm/pkg/job"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

func Run(c cache.Cache, j *job.Job) error {
	return Exec(j.GetShell())
}

func Exec(command []string) error {
	cmd := exec.Command(command[0], command[1:]...)

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
