package config

import (
	"errors"
	"os"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/graph"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"github.com/davecgh/go-spew/spew"
)

func checkErrors(parser *hclparse.Parser, diagnostics hcl.Diagnostics) error {
	if diagnostics.HasErrors() {
		wr := hcl.NewDiagnosticTextWriter(os.Stderr, parser.Files(), 78, true)
		wr.WriteDiagnostics(diagnostics)
		return errors.New("HCL error")
	}
	return nil
}

func DecodeJobs(config string) ([]*job.Job, error) {
	parser := hclparse.NewParser()

	hclFile, diags := parser.ParseHCL([]byte(config), "test")
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	load := struct {
		Jobs []struct {
			Name string `hcl:"name,label"`
			Remain hcl.Body `hcl:",remain"`
		} `hcl:"job,block"`
	}{}

	diags = gohcl.DecodeBody(hclFile.Body, nil, &load)
	if err := checkErrors(parser, diags); err != nil {
		return nil, err
	}

	g := graph.NewJobGraph(nil)

	remains := map[string]hcl.Body{}

	for _, partialJob := range load.Jobs {
		remains[partialJob.Name] = partialJob.Remain

		j := &job.Job{
			Name: partialJob.Name,
		}

		g.AddJob(j)

		attributes, diags := partialJob.Remain.JustAttributes()
		if err := checkErrors(parser, diags); err != nil {
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

	errors := g.IterSorted(func(node *graph.Node) (err error) {
		job := node.Job.(*job.Job)

		context := &hcl.EvalContext{}
		if len(jobMap) > 0 {
			context.Variables = map[string]cty.Value{
				"jobs": cty.MapVal(jobMap),
			}
		}

		spew.Dump(jobMap)

		diags := gohcl.DecodeBody(remains[job.GetName()], context, job)
		if err := checkErrors(parser, diags); err != nil {
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
