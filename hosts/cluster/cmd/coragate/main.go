package main

import (
	"github.com/coragate/coragate/hosts/run"
	"github.com/coragate/coragate/kernel"
)

// Cluster host: default 0.0.0.0:8080 for service / multi-replica listen. Same kernel as the client host.
func main() {
	run.Main(kernel.HostCluster)
}
