package main

import (
	"github.com/coragate/coragate/hosts/run"
	"github.com/coragate/coragate/kernel"
)

// 集群宿主：默认 0.0.0.0:8080，服务/多副本监听。与本机宿主共用同一内核。
func main() {
	run.Main(kernel.HostCluster)
}
