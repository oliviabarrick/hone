package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"os"
	"sort"
)

func sorted(toSort []string) []string {
	sort.Strings(toSort)
	return toSort
}

func TestConfig(t *testing.T) {
	example := `
job "hello" {
	image = "debian:stretch"
	outputs = ["hello"]
	shell = "echo hi > hello"
}

job "world" {
	image = "debian:stretch"
	inputs = jobs.hello.outputs
	shell = "echo world"
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))

	assert.Equal(t, "hello", jobs[0].GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hi > hello"}, jobs[0].GetShell())
	assert.Equal(t, []string{}, jobs[0].GetInputs())
	assert.Equal(t, []string{"hello"}, jobs[0].GetOutputs())
	assert.Equal(t, []string{}, jobs[0].GetDeps())

	assert.Equal(t, "world", jobs[1].GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo world"}, jobs[1].GetShell())
	assert.Equal(t, []string{"hello"}, jobs[1].GetInputs())
	assert.Equal(t, []string{}, jobs[1].GetOutputs())
	assert.Equal(t, []string{"hello"}, jobs[1].GetDeps())
}

func TestConfigConcatLists(t *testing.T) {
	example := `
job "hello" {
	image = "debian:stretch"
	outputs = ["hello"]
	shell = "echo hi > hello"
}

job "moon" {
	image = "debian:stretch"
	outputs = ["hi"]
	shell = "echo hi > hi"
}


job "world" {
	image = "debian:stretch"
	inputs = concat(jobs.hello.outputs, jobs.moon.outputs)
	shell = "echo world"
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(jobs))

	hello := jobs[0]
	moon := jobs[1]

	if hello.GetName() != "hello" {
		hello = jobs[1]
		moon = jobs[0]
	}

	assert.Equal(t, "hello", hello.GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hi > hello"}, hello.GetShell())
	assert.Equal(t, []string{}, hello.GetInputs())
	assert.Equal(t, []string{"hello"}, hello.GetOutputs())
	assert.Equal(t, []string{}, hello.GetDeps())

	assert.Equal(t, "moon", moon.GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hi > hi"}, moon.GetShell())
	assert.Equal(t, []string{}, moon.GetInputs())
	assert.Equal(t, []string{"hi"}, moon.GetOutputs())
	assert.Equal(t, []string{}, moon.GetDeps())

	assert.Equal(t, "world", jobs[2].GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo world"}, jobs[2].GetShell())
	assert.Equal(t, []string{"hello", "hi"}, jobs[2].GetInputs())
	assert.Equal(t, []string{}, jobs[2].GetOutputs())
	assert.Equal(t, []string{"hello", "moon"}, sorted(jobs[2].GetDeps()))
}

func TestConfigSelfReferential(t *testing.T) {
	example := `
job "moon" {
	image = "debian:stretch"
	outputs = ["hi"]
	shell = "echo hi > ${jobs.moon.outputs[0]}"
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))

	assert.Equal(t, "moon", jobs[0].GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hi > hi"}, jobs[0].GetShell())
}

func TestConfigInvalidJob(t *testing.T) {
	example := `
job "moon" {
	shell = lol
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(jobs))
}

func TestConfigComplexSelfReferential(t *testing.T) {
	example := `
job "moon" {
	image = "debian:stretch"
	outputs = [jobs.moon.image]
	shell = "echo hi > ${jobs.moon.outputs[0]}"
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))

	assert.Equal(t, "moon", jobs[0].GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hi > debian:stretch"}, jobs[0].GetShell())
	assert.Equal(t, []string{"debian:stretch"}, jobs[0].GetOutputs())
}

func TestConfigSelfReferentialEnv(t *testing.T) {
	example := `
job "moon" {
	image = "debian:stretch"
	outputs = ["${jobs.moon.env.HELLO}"]
	env = {
		"HELLO" = "${jobs.moon.image}"
	}
	shell = "echo hi > ${jobs.moon.outputs[0]}"
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))

	assert.Equal(t, "moon", jobs[0].GetName())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hi > debian:stretch"}, jobs[0].GetShell())
}

func TestConfigEnv(t *testing.T) {
	example := `
env = [
    "MY_VAR",
    "OTHER_VAR=hello"
]
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	env, err := parser.DecodeEnv()
	assert.Nil(t, err)

	assert.Equal(t, env["MY_VAR"], "")
	assert.Equal(t, env["OTHER_VAR"], "hello")
	assert.NotEqual(t, env["GIT_BRANCH"], "")
	assert.NotEqual(t, env["GIT_COMMIT"], "")
	assert.NotEqual(t, env["GIT_COMMIT_SHORT"], "")
}

func TestConfigEnvWithJob(t *testing.T) {
	example := `
env = [
    "MY_VAR",
    "OTHER_VAR=hello"
]

job "test" {
    image = "lol"
    env = {
        "HELLO" = "${env.OTHER_VAR}"
    }
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	_, err = parser.DecodeEnv()
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.Nil(t, err)
	assert.Equal(t, len(jobs), 1)
	assert.Equal(t, jobs[0].GetEnv()["HELLO"], "hello")
}

func TestConfigSecrets(t *testing.T) {
	example := `
secrets = [
    "MY_SECRET"
]
`

	os.Setenv("MY_SECRET", "hello")
	defer os.Unsetenv("MY_SECRET")

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	secretsMap, err := parser.DecodeSecrets()
	assert.Nil(t, err)

	assert.Equal(t, secretsMap, map[string]string {
		"MY_SECRET": "hello",
	})
}

func TestConfigSecretsWithJob(t *testing.T) {
	example := `
secrets = [
    "MY_SECRET=hello"
]

job "test" {
    image = "lol"
    env = {
        "HELLO" = "${secrets.MY_SECRET}"
    }
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	_, err = parser.DecodeSecrets()
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs([]JobPartial{})
	assert.Nil(t, err)
	assert.Equal(t, len(jobs), 1)
	assert.Equal(t, jobs[0].GetEnv()["HELLO"], "hello")
}

func TestConfigDecodeConfig(t *testing.T) {
	example := `
secrets = [
    "MY_SECRET=hello"
]

job "test" {
    image = "lol"
    env = {
        "HELLO" = "${secrets.MY_SECRET}"
    }
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	config, err := parser.DecodeConfig()
	assert.Nil(t, err)
	assert.Equal(t, len(config.Jobs), 1)
}

func TestConfigTemplate(t *testing.T) {
	example := `
template "hello" {
	image = "debian:stretch"
}

job "moon" {
	template = "hello"
	shell = "echo hello"
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	templates, err := parser.DecodeTemplates()
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs(templates)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))

	assert.Equal(t, "moon", jobs[0].GetName())
	assert.Equal(t, "debian:stretch", jobs[0].GetImage())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hello"}, jobs[0].GetShell())
}

func TestConfigTemplateNested(t *testing.T) {
	example := `
template "other" {
	image = "debian:jessie"
	inputs = ["lol"]
}

template "hello" {
	image = "debian:stretch"
	template = "other"
}

job "moon" {
	template = "hello"
	shell = "echo hello"
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	templates, err := parser.DecodeTemplates()
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs(templates)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))

	assert.Equal(t, "moon", jobs[0].GetName())
	assert.Equal(t, "debian:stretch", jobs[0].GetImage())
	assert.Equal(t, []string{"/bin/sh", "-cex", "echo hello"}, jobs[0].GetShell())
	assert.Equal(t, []string{"lol"}, jobs[0].GetInputs())
}

func TestConfigTemplateSelf(t *testing.T) {
	example := `
template "hello" {
	inputs = [format("%s-%s.pkg.tar.xz", self.name, self.env.VERSION)]
}

job "moon" {
	template = "hello"

	env = {
		"VERSION" = "1234"
	}

	shell = "cat ${self.inputs[0]}"
}

job "world" {
	template = "hello"

	deps = ["moon"]

	env = {
		"VERSION" = "9876"
	}
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	templates, err := parser.DecodeTemplates()
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs(templates)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))

	assert.Equal(t, "moon", jobs[0].GetName())
	assert.Equal(t, []string{"moon-1234.pkg.tar.xz"}, jobs[0].GetInputs())
	assert.Equal(t, []string{"/bin/sh", "-cex", "cat moon-1234.pkg.tar.xz"}, jobs[0].GetShell())

	assert.Equal(t, "world", jobs[1].GetName())
	assert.Equal(t, []string{"world-9876.pkg.tar.xz"}, jobs[1].GetInputs())
}

func TestConfigTemplateDefault(t *testing.T) {
	example := `
template "default" {
	image = "debian:stretch"
}

job "moon" {
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	templates, err := parser.DecodeTemplates()
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs(templates)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))

	assert.Equal(t, "moon", jobs[0].GetName())
	assert.Equal(t, "debian:stretch", jobs[0].GetImage())
}

func TestConfigTemplateImplicitDepend(t *testing.T) {
	example := `
template "hello" {
	outputs = [self.name]
}

job "world" {
	template = "hello"
}

job "other" {
	deps = ["world"]

	inputs = [
		for dep in jobs.other.deps: jobs[dep].outputs[0]
	]
}
`

	parser := NewParser()
	err := parser.Parse(example)
	assert.Nil(t, err)

	templates, err := parser.DecodeTemplates()
	assert.Nil(t, err)

	jobs, err := parser.DecodeJobs(templates)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))

	assert.Equal(t, "world", jobs[0].GetName())
	assert.Equal(t, []string{"world"}, jobs[0].GetOutputs())

	assert.Equal(t, "other", jobs[1].GetName())
	assert.Equal(t, []string{"world"}, jobs[1].GetInputs())
}
