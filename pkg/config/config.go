package config

import (
	"fmt"
	"github.com/hashicorp/hcl"
	"hash/crc32"
	"io/ioutil"
	"strings"
)

func Crc(identifier string) int64 {
	crcTable := crc32.MakeTable(0xD5828281)
	result := int64(crc32.Checksum([]byte(identifier), crcTable))
	return result
}

type Job struct {
	Name    string
	Image   string
	Inputs  []string
	Input   string
	Outputs map[string]string
	Output  string
	Env     map[string]string
	Shell   string
	Deps    []string
}

func (j Job) ID() int64 {
	return Crc(j.Name)
}

func Unmarshal(fname string) (map[string]*Job, error) {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	jobs := map[string]map[string]*Job{} // map[string]*Job{}

	err = hcl.Unmarshal(data, &jobs)

	for name, job := range jobs["job"] {
		job.Name = name
		if !strings.Contains(job.Image, ":") {
			job.Image = fmt.Sprintf("%s:latest", job.Image)
		}
	}

	return jobs["job"], nil
}
