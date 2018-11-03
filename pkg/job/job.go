package job

import (
	"github.com/justinbarrick/farm/pkg/utils"
)

type Job struct {
	Name    string             `hcl:"name,label"`
	Image   string             `hcl:"image"`
	Shell   string             `hcl:"shell"`
	Inputs  *[]string          `hcl:"inputs"`
	Input   *string            `hcl:"input"`
	Outputs *[]string `hcl:"outputs"`
	Output  *string            `hcl:"output"`
	Env     *map[string]string `hcl:"env"`
	Deps    *[]string          `hcl:"deps"`
	Error   error
}

func (j Job) ID() int64 {
	return utils.Crc(j.Name)
}
