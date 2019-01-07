package node

import (
	"github.com/justinbarrick/hone/pkg/utils"
)

type Node interface {
	GetName() string
	GetDeps() []string
	AddDep(string)
	GetError() error
	SetError(error)
	SetDetach(chan bool)
	SetStop(chan bool)
	GetDone() chan bool
	ID() int64
}

func ID(node Node) int64 {
	return utils.Crc(node.GetName())
}
