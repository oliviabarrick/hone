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
	inputs = ["${jobs.hello.outputs}"]
	shell = "echo world"
}
`

	jobs, err := DecodeJobs(example)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))

	assert.Equal(t, "hello", jobs[0].GetName())
	assert.Equal(t, []string{"/bin/sh", "-c", "echo hi > hello"}, jobs[0].GetShell())
	assert.Equal(t, []string{}, jobs[0].GetInputs())
	assert.Equal(t, []string{"hello"}, jobs[0].GetOutputs())

	assert.Equal(t, "world", jobs[1].GetName())
	assert.Equal(t, []string{"/bin/sh", "-c", "echo world"}, jobs[1].GetShell())
	assert.Equal(t, []string{"hello"}, jobs[1].GetInputs())
	assert.Equal(t, []string{}, jobs[1].GetOutputs())
}
