package config

import (
	"errors"
	"os"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/graph"
	"github.com/justinbarrick/hone/pkg/graph/node"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/davecgh/go-spew/spew"
)

type Remains interface {
	GetRemain() hcl.Body
}

type LoadJobs struct {
	Jobs []struct {
		Name string `hcl:"name,label"`
		Remain hcl.Body `hcl:",remain"`
	} `hcl:"job,block"`
	Remain hcl.Body `hcl:",remain"`
}

func (l LoadJobs) GetRemain() hcl.Body {
	return l.Remain
}

type Parser struct {
	parser *hclparse.Parser
	remain hcl.Body
}

func NewParser() Parser {
	return Parser{
		parser: hclparse.NewParser(),
	}
}

func (p *Parser) Parse(config string) error {
	hclFile, diags := p.parser.ParseHCL([]byte(config), "test")
	p.remain = hclFile.Body
	return p.checkErrors(diags)
}

func (p *Parser) checkErrors(diagnostics hcl.Diagnostics) error {
	if diagnostics.HasErrors() {
		wr := hcl.NewDiagnosticTextWriter(os.Stderr, p.parser.Files(), 78, true)
		wr.WriteDiagnostics(diagnostics)
		return errors.New("HCL error")
	}
	return nil
}

func (p *Parser) DecodeBody(body hcl.Body, val interface{}, ctx *hcl.EvalContext) error {
	if ctx == nil {
		ctx = &hcl.EvalContext{}
	}

	if ctx.Functions == nil {
		ctx.Functions = map[string]function.Function{}
	}

	ctx.Functions["concat"] = stdlib.ConcatFunc

	return p.checkErrors(gohcl.DecodeBody(body, ctx, val))
}

func (p *Parser) DecodeRemains(val Remains, ctx *hcl.EvalContext) error {
	err := p.DecodeBody(p.remain, val, ctx)

	p.remain = val.GetRemain()

	return err
}

func (p *Parser) DecodeJobs() ([]*job.Job, error) {
	load := LoadJobs{}

	if err := p.DecodeRemains(&load, nil); err != nil {
		return nil, err
	}

	g := graph.NewGraph(nil)

	remains := map[string]hcl.Body{}

	for _, partialJob := range load.Jobs {
		remains[partialJob.Name] = partialJob.Remain

		j := &job.Job{
			Name: partialJob.Name,
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

				depName, ok := variable[1].(hcl.TraverseAttr)
				if ! ok {
					continue
				}

				g.AddDep(j, depName.Name)
			}
		}
	}

	jobs := []*job.Job{}
	jobMap := map[string]cty.Value{}

	errors := g.IterSorted(func(node node.Node) (err error) {
		job := node.(*job.Job)

		context := &hcl.EvalContext{}
		if len(jobMap) > 0 {
			context.Variables = map[string]cty.Value{
				"jobs": cty.MapVal(jobMap),
			}
		}

		if err := p.DecodeBody(remains[job.GetName()], job, context); err != nil {
			return err
		}

		outputs, err := gocty.ToCtyValue(job.Outputs, cty.List(cty.String))
		if err != nil {
			return err
		}

		jobMap[job.Name] = cty.ObjectVal(map[string]cty.Value{
			"outputs": outputs,
		})
		if err != nil {
			return err
		}

		jobs = append(jobs, job)
		return nil
	})
	if len(errors) > 0 {
		return nil, errors[0]
	}

	spew.Dump(jobs)
	return jobs, nil
}
