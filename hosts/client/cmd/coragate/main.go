package main

import (
	"github.com/coragate/coragate/hosts/run"
	"github.com/coragate/coragate/kernel"
)

// Client host: default 127.0.0.1:8080, colocated with the caller. Same kernel as the cluster host.
func main() {
	run.Main(kernel.HostClient)
}
