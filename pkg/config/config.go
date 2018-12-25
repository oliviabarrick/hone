package config

import (
	"errors"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/git"
	"github.com/justinbarrick/hone/pkg/cache/file"
	"github.com/justinbarrick/hone/pkg/config/types"
	"github.com/justinbarrick/hone/pkg/secrets/vault"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/zclconf/go-cty/cty"
	"os"
	"strings"
)

type FirstLoad struct {
	Env     *[]string `hcl:"env"`
	Secrets *[]string `hcl:"secrets"`
	Remain  hcl.Body  `hcl:",remain"`
}

type SecondLoad struct {
	Workspace *string      `hcl:"workspace"`
	Vault     *vault.Vault `hcl:"vault,block"`
	Remain    hcl.Body     `hcl:",remain"`
}

type ThirdLoad struct {
	Templates []*job.Job   `hcl:"template,block"`
	Remain    hcl.Body     `hcl:",remain"`
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
	config := &types.Config{
		Env: map[string]interface{}{},
	}

	parser := hclparse.NewParser()

	hclFile, diags := parser.ParseHCLFile(fname)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	environ := map[string]cty.Value{}

	fl := &FirstLoad{}
	diags = gohcl.DecodeBody(hclFile.Body, nil, fl)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

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
			config.Env[env[0]] = val
		}
	}

	if repo, err := git.NewRepository(); err == nil {
		for key, value := range repo.GitEnv() {
			environ[key] = cty.StringVal(value)
			config.Env[key] = value
		}
	} else {
		logger.Printf("Failed to load git environment: %s", err)
	}

	variables := map[string]cty.Value{}
	if len(environ) != 0 {
		variables["environ"] = cty.MapVal(environ)
	}

	ctx := hcl.EvalContext{
		Variables: variables,
	}

	sl := &SecondLoad{}
	diags = gohcl.DecodeBody(fl.Remain, &ctx, sl)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	if fl.Secrets != nil {
		workspace := "default"
		if sl.Workspace != nil {
			workspace = *sl.Workspace
		}

		if sl.Vault != nil && sl.Vault.Token != "" {
			err := sl.Vault.Init()
			if err != nil {
				return nil, err
			}

			secrets, err := sl.Vault.LoadSecrets(workspace, *fl.Secrets)
			if err != nil {
				return nil, err
			}
			for key, value := range secrets {
				environ[key] = cty.StringVal(value)
			}
		}
	}

	if len(environ) != 0 {
		variables["environ"] = cty.MapVal(environ)
	}

	ctx = hcl.EvalContext{
		Variables: variables,
	}

	tl := &ThirdLoad{}
	diags = gohcl.DecodeBody(sl.Remain, &ctx, tl)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	diags = gohcl.DecodeBody(tl.Remain, &ctx, config)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	if config.Cache == nil {
		config.Cache = &types.CacheConfig{}
	}

	if config.Cache.File == nil {
		config.Cache.File = &filecache.FileCache{}
	}

	if err := config.RenderTemplates(tl.Templates); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}
