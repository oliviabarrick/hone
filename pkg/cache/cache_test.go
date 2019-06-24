package cache

import (
	"testing"

	"github.com/justinbarrick/hone/pkg/job"
	"github.com/stretchr/testify/assert"
)

func TestHashJob(t *testing.T) {
	expected := "7b8bc47735cb57cce88991bc559dc50f62db66cc26e42a593a573ac76464c326"

	j1 := &job.Job{
		Name: "hello",
		Deps: &job.StringSet{
			"hello",
			"world",
		},
	}

	j2 := &job.Job{
		Name: "hello",
		Deps: &job.StringSet{
			"world",
			"hello",
			"hello",
		},
	}

	assert.Equal(t, j1.Deps.Strings(), j2.Deps.Strings())

	hash, err := HashJob(j1)
	assert.Nil(t, err)
	assert.Equal(t, hash, expected)

	hash, err = HashJob(j2)
	assert.Nil(t, err)
	assert.Equal(t, hash, expected)
}
