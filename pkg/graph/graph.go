package graph

import (
	"errors"
	"fmt"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"github.com/justinbarrick/farm/pkg/config"
)

type JobGraph struct {
	graph *simple.DirectedGraph
	root *config.Job
}

func NewJobGraph(jobs map[string]*config.Job) JobGraph {
	graph := JobGraph{
		graph: simple.NewDirectedGraph(),
		root: &config.Job{
			Name: "ROOT",
		},
	}

	graph.graph.AddNode(graph.root)
	graph.BuildGraph(jobs)
	return graph
}

func (j *JobGraph) BuildGraph(jobs map[string]*config.Job) {
	for _, job := range jobs {
		if j.graph.Node(job.ID()) == nil {
			j.graph.AddNode(job)
		}

/*
		if len(job.Deps) == 0 {
			fmt.Printf("%s -> %s\n", j.root.Name, job.Name)
			j.graph.SetEdge(simple.Edge{
				T: job,
				F: j.root,
			})
		}
*/

		for _, dep := range job.Deps {
//			fmt.Printf("%s -> %s\n", jobs[dep].Name, job.Name)
			j.graph.SetEdge(simple.Edge{
				T: job,
				F: jobs[dep],
			})
		}
	}
}

func (j *JobGraph) ResolveTarget(target string, callback func (config.Job) error) error {
	targetId := config.Crc(target)
	targetNode := j.graph.Node(targetId)
	if targetNode == nil {
		return errors.New(fmt.Sprintf("Target %s not found.", target))
	}

	sorted, err := topo.Sort(j.graph)
	if err != nil {
		return err
	}

	for _, node := range sorted {
		if ! topo.PathExistsIn(j.graph, node, targetNode) {
			continue
		}

		if err := callback(*node.(*config.Job)); err != nil {
			return err
		}
	}

	return nil
}
