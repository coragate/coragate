package main

import (
	"github.com/coragate/coragate/hosts/run"
	"github.com/coragate/coragate/kernel"
)

func main() {
	run.Main(kernel.HostCluster)
}
