package graph

import (
	"errors"
	"fmt"
	"sync"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"github.com/justinbarrick/farm/pkg/config"
)

type JobGraph struct {
	graph *simple.DirectedGraph
}

type Node struct {
	Job  *config.Job
	Done chan bool
}

func NewNode(job *config.Job) *Node {
	return &Node{
		Job: job,
		Done: make(chan bool),
	}
}

func (n Node) ID() int64 {
	return n.Job.ID()
}

func NewJobGraph(jobs map[string]*config.Job) JobGraph {
	graph := JobGraph{
		graph: simple.NewDirectedGraph(),
	}

	graph.BuildGraph(jobs)
	return graph
}

func (j *JobGraph) BuildGraph(jobs map[string]*config.Job) {
	for _, job := range jobs {
		if j.graph.Node(job.ID()) == nil {
			j.graph.AddNode(NewNode(job))
		}

		for _, dep := range job.Deps {
			depJob := jobs[dep]

			if j.graph.Node(depJob.ID()) == nil {
				j.graph.AddNode(NewNode(depJob))
			}

			j.graph.SetEdge(simple.Edge{
				T: j.graph.Node(job.ID()),
				F: j.graph.Node(depJob.ID()),
			})
		}
	}
}

func (j *JobGraph) WaitForDeps(n *Node, callback func (config.Job) error) func (config.Job) error {
	return func (job config.Job) error {
		defer close(n.Done)

		for _, node := range graph.NodesOf(j.graph.To(n.ID())) {
			_ = <-node.(*Node).Done
		}

		return callback(job)
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

	var wg sync.WaitGroup

	for _, node := range sorted {
		if ! topo.PathExistsIn(j.graph, node, targetNode) {
			continue
		}

		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			cb := j.WaitForDeps(n, callback)
			cb(*n.Job)
		}(node.(*Node))
	}

	wg.Wait()
	return nil
}
