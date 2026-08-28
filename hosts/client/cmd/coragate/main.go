package main

import (
	"github.com/coragate/coragate/hosts/run"
	"github.com/coragate/coragate/kernel"
)

// 本机宿主：默认 127.0.0.1:8080，跟调用方同机。与集群宿主共用同一内核。
func main() {
	run.Main(kernel.HostClient)
}
