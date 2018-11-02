package config

import (
	"errors"
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"hash/crc32"
	"os"
	"strings"
)

func Crc(identifier string) int64 {
	crcTable := crc32.MakeTable(0xD5828281)
	result := int64(crc32.Checksum([]byte(identifier), crcTable))
	return result
}

type Config struct {
	Jobs []*Job `hcl:"job,block"`
}

type Job struct {
	Name    string             `hcl:"name,label"`
	Image   string             `hcl:"image"`
	Shell   string             `hcl:"shell"`
	Inputs  *[]string          `hcl:"inputs"`
	Input   *string            `hcl:"input"`
	Outputs *map[string]string `hcl:"outputs"`
	Output  *string            `hcl:"output"`
	Env     *map[string]string `hcl:"env"`
	Deps    *[]string          `hcl:"deps"`
}

func (j Job) ID() int64 {
	return Crc(j.Name)
}

func Unmarshal(fname string) (map[string]*Job, error) {
	parser := hclparse.NewParser()
	hclFile, hclDiagnostics := parser.ParseHCLFile(fname)

	config := Config{}
	moreDiags := gohcl.DecodeBody(hclFile.Body, nil, &config)
	hclDiagnostics = append(hclDiagnostics, moreDiags...)
	if hclDiagnostics.HasErrors() {
		wr := hcl.NewDiagnosticTextWriter(
			os.Stdout,      // writer to send messages to
			parser.Files(), // the parser's file cache, for source snippets
			78,             // wrapping width
			true,           // generate colored/highlighted output
		)

		wr.WriteDiagnostics(hclDiagnostics)
		return nil, errors.New("HCL error")
	}

	jobsMap := map[string]*Job{}
	for _, job := range config.Jobs {
		if !strings.Contains(job.Image, ":") {
			job.Image = fmt.Sprintf("%s:latest", job.Image)
		}
		jobsMap[job.Name] = job
	}

	return jobsMap, nil
}
