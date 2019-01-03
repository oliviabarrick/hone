package config

import (
	"errors"
	"os"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/justinbarrick/hone/pkg/job"
//	"github.com/davecgh/go-spew/spew"
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

	for _, job := range load.Jobs {
		attributes, diags := job.Remain.JustAttributes()
		if err := checkErrors(parser, diags); err != nil {
			return nil, err
		}

		for _, attr := range attributes {
			variables := attr.Expr.Variables()
			for _, variable := range variables {
				variable.TraverseAbs()
			}
		}
	}

	return []*job.Job{}, nil
}
