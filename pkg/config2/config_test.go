package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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

	jobs, err := parser.DecodeJobs()
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

	jobs, err := parser.DecodeJobs()
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
