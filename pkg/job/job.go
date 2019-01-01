
package job

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/justinbarrick/hone/pkg/utils"
	"strings"
)

type Job struct {
	Name    string             `hcl:"name,label" json:"name"`
	Template *string `hcl:"template" hash:"-" json:"-"`
	Image   *string            `hcl:"image" json:"image"`
	Shell   *string            `hcl:"shell" json:"shell"`
	Exec    *[]string          `hcl:"exec" json:"exec"`
	Inputs  *[]string          `hcl:"inputs" json:"inputs"`
	Outputs *[]string          `hcl:"outputs" json:"outputs"`
	Env     *map[string]string `hcl:"env" json:"-"`
	Deps    *[]string          `hcl:"deps" json:"deps"`
	Engine  *string            `hcl:"engine" json:"engine" hash:"-"`
	Condition *string          `hcl:"condition" json:"condition"`
	Privileged *bool           `hcl:"privileged" json:"privileged"`
	Workdir *string            `hcl:"workdir" json:"workdir"`
	Service bool               `hash:"-" json:"service"`
	Error   error              `hash:"-" json:"error"`
	Cached  bool               `hash:"-" json:"cached"`
	Hash         string        `hash:"-" json:"hash"`
	OutputHashes map[string]string      `hash:"-" json:"outputHashes"`
	Detach  chan bool          `hash:"-" json:"-"`
	Stop    chan bool          `hash:"-" json:"-"`
}

func (j *Job) Default(def Job) {
	if j.Image == nil {
		j.Image = def.Image
	}

	if j.Shell == nil {
		j.Shell = def.Shell
	}

	if j.Exec == nil {
		j.Exec = def.Exec
	}

	if j.Inputs == nil {
		j.Inputs = def.Inputs
	}

	if j.Outputs == nil {
		j.Outputs = def.Outputs
	}

	if j.Engine == nil {
		j.Engine = def.Engine
	}

	if j.Deps == nil {
		j.Deps = def.Deps
	}

	if def.Env != nil {
		if j.Env == nil {
			j.Env = def.Env
		} else {
			env := *j.Env

			for key, value := range *def.Env {
				if env[key] != "" {
					continue
				}

				env[key] = value
			}

			j.Env = &env
		}
	}
}

func (j Job) Validate(engine string) error {
	myEngine := j.GetEngine()
	if myEngine == "" {
		myEngine = engine
	}

	if j.Image == nil && myEngine != "local" {
		return errors.New("Image is required when engine is not local.")
	}

	if j.Shell != nil && j.Exec != nil {
		return errors.New("Shell and exec are mutually exclusive.")
	}

	if j.Shell == nil && j.Exec == nil {
		return errors.New("One of shell or exec must be specified.")
	}

	return nil
}

func (j Job) ID() int64 {
	return utils.Crc(j.GetName())
}

func (j Job) GetName() string {
	return j.Name
}

func (j Job) GetImage() string {
	image := *j.Image

	if !strings.Contains(image, ":") {
		image = fmt.Sprintf("%s:latest", image)
	}

	return image
}

func (j Job) GetOutputs() []string {
	outputs := []string{}

	if j.Outputs != nil {
		outputs = *j.Outputs
	}

	return outputs
}

func (j Job) GetInputs() []string {
	inputs := []string{}

	if j.Inputs != nil {
		inputs = *j.Inputs
	}

	return inputs
}

func (j Job) GetShell() []string {
	if j.Exec != nil {
		return *j.Exec
	} else if j.Shell != nil {
		return []string{
			"/bin/sh", "-cex", *j.Shell,
		}
	} else {
		return nil
	}
}

func (j Job) GetEngine() string {
	if j.Engine != nil {
		return *j.Engine
	} else {
		return ""
	}
}

func (j Job) GetEnv() map[string]string {
	if j.Env == nil {
		return map[string]string{}
	}
	return *j.Env
}

func (j Job) GetWorkdir() string {
	if j.Workdir == nil {
		return ""
	}
	return *j.Workdir
}


func (j Job) IsPrivileged() bool {
	if j.Privileged == nil {
		return false
	}

	return *j.Privileged
}

func (j Job) MarshalJSON() ([]byte, error) {
	deps := []string{}
	if j.Deps != nil {
		deps = *j.Deps
	}

	condition := ""
	if j.Condition != nil {
		condition = *j.Condition
	}

	privileged := false
	if j.Privileged != nil {
		privileged = *j.Privileged
	}

	errMsg := ""
	if j.Error != nil {
		errMsg = j.Error.Error()
	}

	return json.Marshal(struct {
		Name string
		Image string
		Shell []string
		Inputs []string
		Outputs []string
		Deps []string
		Engine string
		Condition string
		Privileged bool
		Service bool
		Successful bool
		Error string
		Cached bool
		Hash string
		OutputHashes map[string]string
	}{
		Name: j.GetName(),
		Image: j.GetImage(),
		Shell: j.GetShell(),
		Inputs: j.GetInputs(),
		Outputs: j.GetOutputs(),
		Deps: deps,
		Engine: j.GetEngine(),
		Condition: condition,
		Privileged: privileged,
		Service: j.Service,
		Successful: (j.Error == nil),
		Error: errMsg,
		Cached: j.Cached,
		Hash: j.Hash,
		OutputHashes: j.OutputHashes,
	})
}
