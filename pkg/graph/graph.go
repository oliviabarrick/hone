package graph

import (
	"errors"
	"fmt"
	. "github.com/justinbarrick/hone/pkg/graph/node"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/justinbarrick/hone/pkg/utils"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"sync"
)

type Graph struct {
	graph *simple.DirectedGraph
}

func NewGraph(nodes []Node) Graph {
	graph := Graph{
		graph: simple.NewDirectedGraph(),
	}

	graph.BuildGraph(nodes)
	return graph
}

func (g *Graph) setEdges() error {
	nodes := g.graph.Nodes()

	for _, node := range graph.NodesOf(nodes) {
		for _, dep := range node.(Node).GetDeps() {
			g.graph.SetEdge(simple.Edge{
				T: node,
				F: g.graph.Node(utils.Crc(dep)),
			})
		}
	}

	return nil
}

func (g *Graph) AddDep(targetNode Node, dep string) error {
	node := g.graph.Node(ID(targetNode))
	if node == nil {
		return fmt.Errorf("Node not in graph.")
	}

	node.(Node).AddDep(dep)
	return nil
}

func (g *Graph) AddNode(node Node) {
	g.graph.AddNode(node)
}

func (g *Graph) BuildGraph(nodes []Node) {
	if nodes == nil {
		return
	}

	for _, node := range nodes {
		g.AddNode(node)
	}
}

func (g *Graph) WaitForDeps(callback func(Node) error, servicesWg *sync.WaitGroup) func(Node) error {
	return func(n Node) error {
		defer close(n.GetDone())

		failedDeps := []string{}

		for _, node := range graph.NodesOf(g.graph.To(n.ID())) {
			d := node.(Node)
			_ = <-d.GetDone()
			if d.GetError() != nil {
				failedDeps = append(failedDeps, d.GetName())
			}
		}

		if len(failedDeps) > 0 {
			n.SetError(fmt.Errorf("Failed dependencies: %s", failedDeps))
			logger.LogError(n, n.GetError().Error())
		}

		if n.GetError() != nil {
			return n.GetError()
		}

		servicesWg.Add(1)
		detach := make(chan bool)
		n.SetDetach(detach)

		go func() {
			defer close(detach)
			defer servicesWg.Done()
			n.SetError(callback(n))
		}()

		_ = <-detach

		return n.GetError()
	}
}

func (g *Graph) IterSorted(callback func(Node) error) []error {
	g.setEdges()

	sorted, err := topo.Sort(g.graph)
	if err != nil {
		return []error{err}
	}

	errors := []error{}
	for _, node := range sorted {
		err := callback(node.(Node))
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (g *Graph) IterTarget(target string, callback func(Node) error) []error {
	targetId := utils.Crc(target)
	targetNode := g.graph.Node(targetId)
	if targetNode == nil {
		return []error{errors.New(fmt.Sprintf("Target %s not found.", target))}
	}

	return g.IterSorted(func(node Node) error {
		if !topo.PathExistsIn(g.graph, g.graph.Node(node.ID()), targetNode) {
			return nil
		}

		return callback(node)
	})
}

func (g *Graph) ResolveTarget(target string, callback func(Node) error) []error {
	stopCh := make(chan bool)

	var wg sync.WaitGroup
	var servicesWg sync.WaitGroup

	callback = g.WaitForDeps(callback, &servicesWg)
	errors := []error{}

	iterErrors := g.IterTarget(target, func(node Node) error {
		wg.Add(1)

		go func(n Node) {
			defer wg.Done()
			n.SetStop(stopCh)
			err := callback(n)
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

func (g *Graph) LongestTarget(target string) (int, []error) {
	longestName := 0
	lock := sync.Mutex{}

	errors := g.IterTarget(target, func(n Node) error {
		lock.Lock()

		name := n.GetName()

		if len(name) > longestName {
			longestName = len(name)
		}

		lock.Unlock()
		return nil
	})

	return longestName, errors
}
