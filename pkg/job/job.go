package job

import (
	"github.com/justinbarrick/farm/pkg/utils"
	"strings"
	"errors"
	"fmt"
)

type Job struct {
	Name    string             `hcl:"name,label"`
	Image   string             `hcl:"image"`
	Shell   *string             `hcl:"shell"`
	Exec    *[]string             `hcl:"exec"`
	Inputs  *[]string          `hcl:"inputs"`
	Input   *string            `hcl:"input"`
	Outputs *[]string          `hcl:"outputs"`
	Output  *string            `hcl:"output"`
	Env     *map[string]string `hcl:"env"`
	Deps    *[]string          `hcl:"deps"`
	Engine *string `hcl:"engine",hash:"-"`
	Error   error `hash:"-"`
}

func (j Job) Validate(engine string) error {
	myEngine := j.GetEngine()
	if myEngine == "" {
		myEngine = engine
	}

	if j.Image == "" && engine != "local" {
		return errors.New("Image is required when engine is not local.")
	}

	if j.Shell != nil && j.Exec != nil {
		return errors.New("Shell and exec are mutually exclusive.")
	}

	return nil
}

func (j Job) ID() int64 {
	return utils.Crc(j.Name)
}

func (j Job) GetImage() string {
    if ! strings.Contains(j.Image, ":") {
        j.Image = fmt.Sprintf("%s:latest", j.Image)
    }

    return j.Image
}

func (j Job) GetOutputs() []string {
    outputs := []string{}

    if j.Outputs != nil {
        outputs = *j.Outputs
    }

    if j.Output != nil {
        outputs = append(outputs, *j.Output)
    }

    return outputs
}

func (j Job) GetInputs() []string {
    inputs := []string{}

    if j.Inputs != nil {
        inputs = *j.Inputs
    }

    if j.Input != nil {
        inputs = append(inputs, *j.Input)
    }

    return inputs
}

func (j Job) GetShell() []string {
    if j.Exec != nil {
        return *j.Exec
    } else {
        return []string{
            "/bin/sh", "-cex", *j.Shell,
        }
    }
}

func (j Job) GetEngine() string {
	if j.Engine != nil {
		return *j.Engine
	} else {
		return ""
	}
}
