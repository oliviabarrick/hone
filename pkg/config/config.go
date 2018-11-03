package config

import (
	"errors"
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/justinbarrick/farm/pkg/cache/file"
	"github.com/justinbarrick/farm/pkg/config/types"
	"github.com/zclconf/go-cty/cty"
	"os"
	"strings"
)

type FirstLoad struct {
	Env    *[]string `hcl:"env"`
	Remain hcl.Body  `hcl:",remain"`
}

func checkErrors(parser *hclparse.Parser, diagnostics hcl.Diagnostics) error {
	if diagnostics.HasErrors() {
		wr := hcl.NewDiagnosticTextWriter(os.Stderr, parser.Files(), 78, true)
		wr.WriteDiagnostics(diagnostics)
		return errors.New("HCL error")
	}
	return nil
}

func Unmarshal(fname string) (*types.Config, error) {
	//variables := &Variables{}
	config := &types.Config{}
	parser := hclparse.NewParser()

	hclFile, diags := parser.ParseHCLFile(fname)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	fl := &FirstLoad{}

	diags = gohcl.DecodeBody(hclFile.Body, nil, fl)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	environ := map[string]cty.Value{}
	if fl.Env != nil {
		for _, key := range *fl.Env {
			env := strings.SplitN(key, "=", 2)
			defaultVal := ""
			if len(env) > 1 {
				defaultVal = env[1]
			}
			val := os.Getenv(env[0])
			if val == "" {
				val = defaultVal
			}
			environ[env[0]] = cty.StringVal(val)
		}
	}

	ctx := hcl.EvalContext{
		Variables: map[string]cty.Value{
			"environ": cty.MapVal(environ),
		},
	}

	diags = gohcl.DecodeBody(fl.Remain, &ctx, config)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	for _, job := range config.Jobs {
		if !strings.Contains(job.Image, ":") {
			job.Image = fmt.Sprintf("%s:latest", job.Image)
		}

		if job.Inputs == nil {
			job.Inputs = &[]string{}
		}
		if job.Input != nil {
			*job.Inputs = append(*job.Inputs, *job.Input)
		}

		if job.Outputs == nil {
			job.Outputs = &[]string{}
		}
		if job.Output != nil {
			*job.Outputs = append(*job.Outputs, *job.Output)
		}
	}

	if config.Cache == nil {
		config.Cache = &types.CacheConfig{}
	}

	if config.Cache.File == nil {
		config.Cache.File = &filecache.FileCache{}
	}

	return config, nil
}
