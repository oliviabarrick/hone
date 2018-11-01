package main

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"fmt"
	"log"
	"os"
	"github.com/hashicorp/hcl"
	"hash/crc32"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
//	"gonum.org/v1/gonum/graph/traverse"
	//"gonum.org/v1/gonum/graph/path"
//	"github.com/davecgh/go-spew/spew"
	"github.com/justinbarrick/farm/pkg/config"
	"github.com/justinbarrick/farm/pkg/executors/docker"
	"github.com/justinbarrick/farm/pkg/graph"
)

func main() {
	jobs, err := config.Unmarshal(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	g := graph.NewJobGraph(jobs)
	if err := g.ResolveTarget(os.Args[2], func(j config.Job) error {
		docker.Run(j)
	}); err != nil {
		log.Fatal(err)
	}
}
