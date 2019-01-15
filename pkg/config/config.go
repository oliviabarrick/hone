package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/justinbarrick/hone/pkg/cache/file"
	"github.com/justinbarrick/hone/pkg/config/types"
	"github.com/justinbarrick/hone/pkg/executors/kubernetes"
	"github.com/justinbarrick/hone/pkg/git"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/scm"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/justinbarrick/hone/pkg/graph"
	"github.com/justinbarrick/hone/pkg/graph/node"
	"github.com/justinbarrick/hone/pkg/secrets/vault"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

type Remains interface {
	GetRemain() hcl.Body
}

type Parser struct {
	parser  *hclparse.Parser
	body    hcl.Body
	remain  hcl.Body
	ctx *hcl.EvalContext
}

func NewParser() Parser {
	return Parser{
		parser: hclparse.NewParser(),
	}
}

func (p *Parser) Parse(config string) error {
	hclFile, diags := p.parser.ParseHCL([]byte(config), "test")
	p.remain = hclFile.Body
	p.body = hclFile.Body
	return p.checkErrors(diags)
}

func (p *Parser) ParseFile(path string) error {
	hclFile, diags := p.parser.ParseHCLFile(path)
	p.remain = hclFile.Body
	p.body = hclFile.Body
	return p.checkErrors(diags)
}

func (p *Parser) checkErrors(err error) error {
	switch e := err.(type) {
	case hcl.Diagnostics:
		if e.HasErrors() {
			wr := hcl.NewDiagnosticTextWriter(os.Stderr, p.parser.Files(), 78, true)
			wr.WriteDiagnostics(e)
			return e
		}
		return nil
	}
	return err
}

func (p *Parser) GetContext() *hcl.EvalContext {
	if p.ctx == nil {
		p.ctx = &hcl.EvalContext{}
	}

	if p.ctx.Functions == nil {
		p.ctx.Functions = map[string]function.Function{}
		p.ctx.Functions["not"] = stdlib.NotFunc
		p.ctx.Functions["and"] = stdlib.AndFunc
		p.ctx.Functions["or"] = stdlib.OrFunc
		p.ctx.Functions["bytesLen"] = stdlib.BytesLenFunc
		p.ctx.Functions["bytesSlice"] = stdlib.BytesSliceFunc
		p.ctx.Functions["hasIndex"] = stdlib.HasIndexFunc
		p.ctx.Functions["index"] = stdlib.IndexFunc
		p.ctx.Functions["length"] = stdlib.LengthFunc
		p.ctx.Functions["csvDecode"] = stdlib.CSVDecodeFunc
		p.ctx.Functions["formatDate"] = stdlib.FormatDateFunc
		p.ctx.Functions["format"] = stdlib.FormatFunc
		p.ctx.Functions["formatList"] = stdlib.FormatListFunc
		p.ctx.Functions["equal"] = stdlib.EqualFunc
		p.ctx.Functions["notEqual"] = stdlib.NotEqualFunc
		p.ctx.Functions["coalesce"] = stdlib.CoalesceFunc
		p.ctx.Functions["jsonEncode"] = stdlib.JSONEncodeFunc
		p.ctx.Functions["jsonDecode"] = stdlib.JSONDecodeFunc
		p.ctx.Functions["absolute"] = stdlib.AbsoluteFunc
		p.ctx.Functions["add"] = stdlib.AddFunc
		p.ctx.Functions["subtract"] = stdlib.SubtractFunc
		p.ctx.Functions["multiply"] = stdlib.MultiplyFunc
		p.ctx.Functions["divide"] = stdlib.DivideFunc
		p.ctx.Functions["modulo"] = stdlib.ModuloFunc
		p.ctx.Functions["greaterThan"] = stdlib.GreaterThanFunc
		p.ctx.Functions["greaterThanOrEqualTo"] = stdlib.GreaterThanOrEqualToFunc
		p.ctx.Functions["lessThan"] = stdlib.LessThanFunc
		p.ctx.Functions["lessThanOrEqualTo"] = stdlib.LessThanOrEqualToFunc
		p.ctx.Functions["negate"] = stdlib.NegateFunc
		p.ctx.Functions["min"] = stdlib.MinFunc
		p.ctx.Functions["max"] = stdlib.MaxFunc
		p.ctx.Functions["int"] = stdlib.IntFunc
		p.ctx.Functions["concat"] = stdlib.ConcatFunc
		p.ctx.Functions["hasElement"] = stdlib.SetHasElementFunc
		p.ctx.Functions["union"] = stdlib.SetUnionFunc
		p.ctx.Functions["intersection"] = stdlib.SetIntersectionFunc
		p.ctx.Functions["setSubtract"] = stdlib.SetSubtractFunc
		p.ctx.Functions["diff"] = stdlib.SetSymmetricDifferenceFunc
		p.ctx.Functions["upper"] = stdlib.UpperFunc
		p.ctx.Functions["lower"] = stdlib.LowerFunc
		p.ctx.Functions["reverse"] = stdlib.ReverseFunc
		p.ctx.Functions["strlen"] = stdlib.StrlenFunc
		p.ctx.Functions["substr"] = stdlib.SubstrFunc
		p.ctx.Functions["basename"] = function.New(&function.Spec{
			Params: []function.Parameter{
				{
					Name:             "path",
					Type:             cty.String,
				},
			},
			Type: function.StaticReturnType(cty.String),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				return cty.StringVal(filepath.Base(args[0].AsString())), nil
			},
		})
		p.ctx.Functions["pathjoin"] = function.New(&function.Spec{
			VarParam: &function.Parameter{
				Name:      "paths",
				Type:      cty.String,
			},
			Type: function.StaticReturnType(cty.String),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				paths := []string{}

				for _, arg := range args {
					paths = append(paths, arg.AsString())
				}

				return cty.StringVal(filepath.Join(paths...)), nil
    	},
		})
		p.ctx.Functions["join"] = function.New(&function.Spec{
			Params: []function.Parameter{
				{
					Name:             "strs",
					Type:             cty.List(cty.String),
				},
				{
					Name:             "sep",
					Type:             cty.String,
				},
			},
			Type: function.StaticReturnType(cty.String),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				strs := []string{}

				arg0 := args[0].AsValueSlice()

				for _, arg := range arg0 {
					strs = append(strs, arg.AsString())
				}

				return cty.StringVal(strings.Join(strs, args[1].AsString())), nil
    	},
		})
		p.ctx.Functions["split"] = function.New(&function.Spec{
			Params: []function.Parameter{
				{
					Name:             "str",
					Type:             cty.String,
				},
				{
					Name:             "sep",
					Type:             cty.String,
				},
			},
			Type: function.StaticReturnType(cty.List(cty.String)),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				str := args[0].AsString()
				sep := args[1].AsString()

				split := strings.Split(str, sep)

				return gocty.ToCtyValue(split, cty.List(cty.String))
    	},
		})

	}

	if p.ctx.Variables == nil {
		p.ctx.Variables = map[string]cty.Value{}
	}

	return p.ctx
}

func (p *Parser) Decode(body hcl.Body, val interface{}) error {
	return gohcl.DecodeBody(body, p.GetContext(), val)
}

func (p *Parser) DecodeRemains(val Remains) error {
	err := p.checkErrors(p.Decode(p.remain, val))

	p.remain = val.GetRemain()

	return err
}

func (p *Parser) DecodeBody(val interface{}) error {
	return p.checkErrors(p.Decode(p.body, val))
}

func (p *Parser) DecodeEnv() (map[string]string, error) {
	envMap := map[string]string{}

	envStruct := struct {
		Env *[]string `hcl:"env"`
		Remain hcl.Body `hcl:",remain"`
	}{}

	err := p.DecodeBody(&envStruct)
	if err != nil {
		return envMap, err
	}

	if envStruct.Env != nil {
		for _, key := range *envStruct.Env {
			env := strings.SplitN(key, "=", 2)
			defaultVal := ""
			if len(env) > 1 {
				defaultVal = env[1]
			}
			val := os.Getenv(env[0])
			if val == "" {
				val = defaultVal
			}
			envMap[env[0]] = val
		}
	}

	if repo, err := git.NewRepository(); err == nil {
		for key, value := range repo.GitEnv() {
			envMap[key] = value
		}
	} else {
		logger.Printf("Failed to load git environment: %s", err)
	}

	p.ctx.Variables["env"], err = gocty.ToCtyValue(envMap, cty.Map(cty.String))
	if err != nil {
		return envMap, err
	}

	return envMap, nil
}

func (p *Parser) DecodeSecrets() (map[string]string, error) {
	secretsMap := map[string]string{}

	setSecrets := func() (err error) {
		if len(secretsMap) == 0 {
			p.ctx.Variables["secrets"] = cty.MapValEmpty(cty.String)
			return
		}

		p.ctx.Variables["secrets"], err = gocty.ToCtyValue(secretsMap, cty.Map(cty.String))
		return
	}

	secretsStruct := struct {
		Secrets   *[]string    `hcl:"secrets"`
		Workspace *string      `hcl:"workspace"`
		Vault     *vault.Vault `hcl:"vault,block"`
		Remain    hcl.Body     `hcl:",remain"`
	}{}

	err := p.DecodeBody(&secretsStruct)
	if err != nil {
		return secretsMap, err
	}

	if secretsStruct.Secrets == nil {
		return secretsMap, setSecrets()
	}

	workspace := "default"
	if secretsStruct.Workspace != nil {
		workspace = *secretsStruct.Workspace
	}

	secrets := []string{}
	if secretsStruct.Secrets != nil {
		secrets = *secretsStruct.Secrets
	}

	if secretsStruct.Vault == nil {
		secretsStruct.Vault = &vault.Vault{}
	}

	err = secretsStruct.Vault.Init()
	if err != nil {
		return secretsMap, err
	}

	secretsMap, err = secretsStruct.Vault.LoadSecrets(workspace, secrets)
	if err != nil {
		return secretsMap, err
	}

	return secretsMap, setSecrets()
}

type JobPartial struct {
	Name string     `hcl:"name,label"`
	Deps *[]string `hcl:"deps"`
	Remain hcl.Body `hcl:",remain"`
}

func (p *Parser) DecodeTemplates() ([]JobPartial, error) {
	load := struct {
		Templates []JobPartial `hcl:"template,block"`
		Remain hcl.Body `hcl:",remain"`
	}{}

	if err := p.DecodeBody(&load); err != nil {
		return nil, err
	}

	return load.Templates, nil
}


func (p *Parser) DecodeSCMs() ([]*scm.SCM, error) {
	load := struct {
		Repositories []*scm.SCM `hcl:"repository,block"`
		Remain hcl.Body `hcl:",remain"`
	}{}

	if err := p.DecodeBody(&load); err != nil {
		return nil, err
	}

	return load.Repositories, nil
}

func (p *Parser) DecodeCache() (types.CacheConfig, error) {
	load := struct {
		Cache *types.CacheConfig `hcl:"cache,block"`
		Remain hcl.Body `hcl:",remain"`
	}{}

	if err := p.DecodeBody(&load); err != nil {
		return types.CacheConfig{}, err
	}

	if load.Cache == nil {
		load.Cache = &types.CacheConfig{}
	}

	if load.Cache.File == nil {
		load.Cache.File = &filecache.FileCache{}
	}

	return *load.Cache, nil
}

func (p *Parser) DecodeKubernetes() (*kubernetes.Kubernetes, error) {
	load := struct {
		Kubernetes *kubernetes.Kubernetes `hcl:"kubernetes,block"`
		Remain hcl.Body `hcl:",remain"`
	}{}

	if err := p.DecodeBody(&load); err != nil {
		return nil, err
	}

	return load.Kubernetes, nil
}

func (p *Parser) DecodeEngine() (*string, error) {
	load := struct {
		Engine *string `hcl:"engine"`
		Remain hcl.Body `hcl:",remain"`
	}{}

	if err := p.DecodeBody(&load); err != nil {
		return nil, err
	}

	return load.Engine, nil
}

func (p *Parser) templateForJob(job *job.Job, templates []JobPartial, jobIsTemplate bool) (*JobPartial, error) {
	for _, template := range templates {
		if job.Template != nil && *job.Template == template.Name {
			return &template, nil
		} else if job.Template == nil && template.Name == "default" && ! jobIsTemplate {
			return &template, nil
		}
	}

	if job.Template == nil {
		return nil, nil
	}

	return nil, fmt.Errorf("Template '%s' not found.", *job.Template)
}

func (p *Parser) DecodeJobs(templates []JobPartial) ([]*job.Job, error) {
	load := struct {
		Jobs []JobPartial `hcl:"job,block"`
		Remain hcl.Body `hcl:",remain"`
	}{}

	if err := p.DecodeBody(&load); err != nil {
		return nil, err
	}

	g := graph.NewGraph(nil)

	remains := map[string]hcl.Body{}

	for _, partialJob := range load.Jobs {
		remains[partialJob.Name] = partialJob.Remain

		j := &job.Job{
			Name: partialJob.Name,
		}

		if partialJob.Deps != nil {
			for _, dep := range *partialJob.Deps {
				j.AddDep(dep)
			}
		}

		g.AddNode(j)

		attributes, diags := partialJob.Remain.JustAttributes()
		if err := p.checkErrors(diags); err != nil {
			return nil, err
		}

		for _, attr := range attributes {
			variables := attr.Expr.Variables()
			for _, variable := range variables {
				if variable.RootName() != "jobs" {
					continue
				}

				if len(variable) < 2 {
					continue
				}

				depName, ok := variable[1].(hcl.TraverseAttr)
				if ! ok {
					continue
				}

				g.AddDep(j, depName.Name)
			}
		}
	}

	jobs := []*job.Job{}

	errors := g.IterSorted(func(node node.Node) (err error) {
		j := node.(*job.Job)

		if err := p.decodeJob(j, remains[j.GetName()], 0, templates, false, nil); err != nil {
			return err
		}

		jobs = append(jobs, j)
		return nil
	})
	if len(errors) > 0 {
		return nil, errors[0]
	}

	return jobs, nil
}

func (p *Parser) setJob(j *job.Job) error {
	jobMap := map[string]cty.Value{}
	if ! p.ctx.Variables["jobs"].IsNull() {
		jobMap = p.ctx.Variables["jobs"].AsValueMap()
	}

	jobCty, err := j.ToCty()
	if err != nil {
		return err
	}

	jobMap[j.Name] = jobCty
	p.ctx.Variables["jobs"] = cty.MapVal(jobMap)
	p.ctx.Variables["self"] = jobCty
	return nil
}

func (p *Parser) decodeJob(j *job.Job, body hcl.Body, depth int, templates []JobPartial, jobIsTemplate bool, self *job.Job) error {
	if self == nil {
		self = j
	}

	if err := p.setJob(self); err != nil {
		return err
	}

	decodeErr := p.Decode(body, j)
	e, ok := decodeErr.(hcl.Diagnostics)
	if ok != true {
		return decodeErr
	}

	if err := p.setJob(self); err != nil {
		return err
	}

	if depth > 20 {
		return p.checkErrors(e)
	}

	template, err := p.templateForJob(j, templates, jobIsTemplate)
	if err != nil {
		return err
	}

	if template != nil {
		templateJob := job.Job{
			Name: self.GetName(),
		}

		if err := p.decodeJob(&templateJob, template.Remain, depth + 1, templates, true, self); err != nil {
			return err
		}

		j.Default(templateJob)
	}

	if err := p.setJob(self); err != nil {
		return err
	}

	if e.HasErrors() {
		return p.decodeJob(j, body, depth + 1, templates, jobIsTemplate, self)
	}

	return nil
}

func (p *Parser) DecodeConfig() (config types.Config, err error) {
	if config.Env, err = p.DecodeEnv(); err != nil {
		return
	}

	if config.Secrets, err = p.DecodeSecrets(); err != nil {
		return
	}

	if config.SCM, err = p.DecodeSCMs(); err != nil {
		return
	}

	if config.Cache, err = p.DecodeCache(); err != nil {
		return
	}

	if config.Kubernetes, err = p.DecodeKubernetes(); err != nil {
		return
	}

	if config.Engine, err = p.DecodeEngine(); err != nil {
		return
	}

	templates, err := p.DecodeTemplates()
	if err != nil {
		return
	}

	if config.Jobs, err = p.DecodeJobs(templates); err != nil {
		return
	}

/*
	if err = config.RenderTemplates(templates); err != nil {
		return
	}
*/

	return
}

func Unmarshal(path string) (*types.Config, error) {
	parser := NewParser()

	if err := parser.ParseFile(path); err != nil {
		return nil, err
	}

	config, err := parser.DecodeConfig()
	if err != nil {
		return nil, err
	}

	return &config, nil
}
