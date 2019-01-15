
package job

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"github.com/justinbarrick/hone/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
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
	deps    []string
	Engine  *string            `hcl:"engine" json:"engine" hash:"-"`
	Condition *string          `hcl:"condition" json:"condition"`
	Privileged *bool           `hcl:"privileged" json:"privileged"`
	Workdir *string            `hcl:"workdir" json:"workdir"`
	Service *bool              `hcl:"service" json:"service" hash:"-"`
	Cached  bool               `hash:"-" json:"cached"`
	Hash         string        `hash:"-" json:"hash"`
	OutputHashes map[string]string      `hash:"-" json:"outputHashes"`
	Detach  chan bool          `hash:"-" json:"-"`
	Stop    chan bool          `hash:"-" json:"-"`
	Error   error              `hash:"-" json:"error"`
	done    chan bool          `hash:"-"`
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

	if j.Workdir == nil {
		j.Workdir = def.Workdir
	}

	j.deps = append(j.deps, def.GetDeps()...)

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

func (j Job) GetError() error {
	return j.Error
}

func (j *Job) GetDone() (chan bool) {
	if j.done == nil {
		j.done = make(chan bool)
	}

	return j.done
}

func (j *Job) SetError(err error) {
	j.Error = err
}

func (j *Job) SetStop(stopCh chan bool) {
	j.Stop = stopCh
}

func (j *Job) SetDetach(detachCh chan bool) {
	j.Detach = detachCh
}

func (j Job) GetDeps() []string {
	hclDeps := []string{}

	if j.Deps != nil {
		hclDeps = *j.Deps
	}

	allDeps := map[string]bool{}

	for _, dep := range hclDeps {
		allDeps[dep] = true
	}

	for _, dep := range j.deps {
		allDeps[dep] = true
	}

	strDeps := []string{}
	for key, _ := range allDeps {
		strDeps = append(strDeps, key)
	}

	return strDeps
}

func (j *Job) AddDep(dep string) {
	if dep == j.GetName() {
		return
	}

	for _, oldDep := range j.deps {
		if oldDep == dep {
			return
		}
	}

	j.deps = append(j.deps, dep)
}

func (j Job) IsPrivileged() bool {
	if j.Privileged == nil {
		return false
	}

	return *j.Privileged
}

func (j Job) MarshalJSON() ([]byte, error) {
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
		Deps: j.GetDeps(),
		Engine: j.GetEngine(),
		Condition: condition,
		Privileged: privileged,
		Service: j.IsService(),
		Successful: (j.Error == nil),
		Error: errMsg,
		Cached: j.Cached,
		Hash: j.Hash,
		OutputHashes: j.OutputHashes,
	})
}

func (j *Job) IsService() bool {
	if j.Service == nil {
		return false
	}

	return *j.Service
}

func (j *Job) ID() int64 {
	return utils.Crc(j.GetName())
}

func (j Job) setMapBool(objMap map[string]cty.Value, key string, value *bool) {
	if value != nil {
		objMap[key] = cty.BoolVal(*value)
	} else {
		objMap[key] = cty.NullVal(cty.Bool)
	}
}

func (j Job) setMapString(objMap map[string]cty.Value, key string, value *string) {
	if value != nil {
		objMap[key] = cty.StringVal(*value)
	} else {
		objMap[key] = cty.NullVal(cty.String)
	}
}

func (j Job) setMapStringList(objMap map[string]cty.Value, key string, value *[]string) error {
	if value != nil {
		valueEncoded, err := gocty.ToCtyValue(value, cty.List(cty.String))
		if err != nil {
			return err
		}
		objMap[key] = valueEncoded
	} else {
		objMap[key] = cty.NullVal(cty.List(cty.String))
	}

	return nil
}

func (j Job) setMapStringMap(objMap map[string]cty.Value, key string, value *map[string]string) error {
	if value != nil {
		valueEncoded, err := gocty.ToCtyValue(value, cty.Map(cty.String))
		if err != nil {
			return err
		}
		objMap[key] = valueEncoded
	} else {
		objMap[key] = cty.NullVal(cty.Map(cty.String))
	}

	return nil
}


func (j *Job) ToCty() (cty.Value, error) {
	objMap := map[string]cty.Value{
		"name": cty.StringVal(j.Name),
	}

	j.setMapString(objMap, "image", j.Image)
	j.setMapString(objMap, "shell", j.Shell)
	j.setMapString(objMap, "workdir", j.Workdir)
	j.setMapString(objMap, "condition", j.Condition)
	j.setMapString(objMap, "engine", j.Engine)
	j.setMapBool(objMap, "privileged", j.Privileged)

	if err := j.setMapStringList(objMap, "exec", j.Exec); err != nil {
		return cty.NilVal, err
	}

	if err := j.setMapStringList(objMap, "inputs", j.Inputs); err != nil {
		return cty.NilVal, err
	}

	if err := j.setMapStringList(objMap, "outputs", j.Outputs); err != nil {
		return cty.NilVal, err
	}

	deps := j.GetDeps()
	if err := j.setMapStringList(objMap, "deps", &deps); err != nil {
		return cty.NilVal, err
	}

	if err := j.setMapStringMap(objMap, "env", j.Env); err != nil {
		return cty.NilVal, err
	}

	return cty.ObjectVal(objMap), nil
}
