package graph

import (
	"errors"
	"fmt"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/justinbarrick/hone/pkg/utils"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"sync"
)

type JobGraph struct {
	graph *simple.DirectedGraph
}

func ID(job job.JobInt) int64 {
	return utils.Crc(job.GetName())
}

type Node struct {
	Job  job.JobInt
	Done chan bool
	deps map[string]bool
}

func NewNode(job job.JobInt) *Node {
	node := &Node{
		Job:  job,
		Done: make(chan bool),
		deps: map[string]bool{},
	}

	for _, dep := range job.GetDeps() {
		node.AddDep(dep)
	}

	return node
}

func (n Node) ID() int64 {
	return ID(n.Job)
}

func (n *Node) AddDep(dep string) {
	n.deps[dep] = true
}

func (n Node) GetDeps() []string {
	deps := []string{}

	for dep, _ := range n.deps {
		deps = append(deps, dep)
	}

	return deps
}

func NewJobGraph(jobs []job.JobInt) JobGraph {
	graph := JobGraph{
		graph: simple.NewDirectedGraph(),
	}

	graph.BuildGraph(jobs)
	return graph
}

func (j *JobGraph) setEdges() error {
	nodes := j.graph.Nodes()

	for _, node := range graph.NodesOf(nodes) {
		for _, dep := range node.(*Node).GetDeps() {
			j.graph.SetEdge(simple.Edge{
				T: node,
				F: j.graph.Node(utils.Crc(dep)),
			})
		}
	}

	return nil
}

func (j *JobGraph) AddDep(job job.JobInt, dep string) error {
	node := j.graph.Node(ID(job))
	if node == nil {
		return fmt.Errorf("Job not in graph.")
	}

	node.(*Node).AddDep(dep)
	return nil
}

func (j *JobGraph) AddJob(job job.JobInt) {
	j.graph.AddNode(NewNode(job))
}

func (j *JobGraph) BuildGraph(jobs []job.JobInt) {
	if jobs == nil {
		return
	}

	for _, job := range jobs {
		j.AddJob(job)
	}
}

func (j *JobGraph) WaitForDeps(n *Node, callback func(job.JobInt) error, servicesWg *sync.WaitGroup) func(job.JobInt) error {
	return func(job job.JobInt) error {
		defer close(n.Done)

		failedDeps := []string{}

		for _, node := range graph.NodesOf(j.graph.To(n.ID())) {
			d := node.(*Node)
			_ = <-d.Done
			if d.Job.GetError() != nil {
				failedDeps = append(failedDeps, d.Job.GetName())
			}
		}

		if len(failedDeps) > 0 {
			n.Job.SetError(errors.New(fmt.Sprintf("Failed dependencies: %s", failedDeps)))
			logger.LogError(job, n.Job.GetError().Error())
		}

		if n.Job.GetError() != nil {
			return n.Job.GetError()
		}

		servicesWg.Add(1)
		detach := make(chan bool)
		job.SetDetach(detach)

		go func() {
			defer close(detach)
			defer servicesWg.Done()
			n.Job.SetError(callback(job))
		}()

		_ = <-detach

		return n.Job.GetError()
	}
}

func (j *JobGraph) IterSorted(callback func(*Node) error) []error {
	j.setEdges()

	sorted, err := topo.Sort(j.graph)
	if err != nil {
		return []error{err}
	}

	errors := []error{}
	for _, node := range sorted {
		err := callback(node.(*Node))
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (j *JobGraph) IterTarget(target string, callback func(*Node) error) []error {
	targetId := utils.Crc(target)
	targetNode := j.graph.Node(targetId)
	if targetNode == nil {
		return []error{errors.New(fmt.Sprintf("Target %s not found.", target))}
	}

	return j.IterSorted(func(node *Node) error {
		if !topo.PathExistsIn(j.graph, j.graph.Node(node.ID()), targetNode) {
			return nil
		}

		return callback(node)
	})
}

func (j *JobGraph) ResolveTarget(target string, callback func(job.JobInt) error) []error {
	stopCh := make(chan bool)

	var wg sync.WaitGroup
	var servicesWg sync.WaitGroup

	errors := []error{}

	iterErrors := j.IterTarget(target, func(node *Node) error {
		wg.Add(1)

		go func(n *Node) {
			defer wg.Done()
			cb := j.WaitForDeps(n, callback, &servicesWg)
			n.Job.SetStop(stopCh)
			err := cb(n.Job)
			if err != nil {
				errors = append(errors, err)
			}
		}(node)

		return nil
	})

	errors = append(errors, iterErrors...)

	wg.Wait()
	close(stopCh)
	servicesWg.Wait()
	return errors
}

func (j *JobGraph) LongestTarget(target string) (int, []error) {
	longestJob := 0
	lock := sync.Mutex{}

	errors := j.IterTarget(target, func(n *Node) error {
		lock.Lock()

		name := n.Job.GetName()

		if len(name) > longestJob {
			longestJob = len(name)
		}

		lock.Unlock()
		return nil
	})

	return longestJob, errors
}
