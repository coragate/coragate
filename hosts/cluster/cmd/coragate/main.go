package main

import (
	"fmt"
	"os"
)

// 集群宿主：默认服务监听（0.0.0.0）。代理逻辑见 SPEC-gateway-mvp T1。
func main() {
	fmt.Fprintln(os.Stderr, "coragate cluster host: 默认 0.0.0.0，尚未代理（见 T1）。规范：https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp")
	os.Exit(0)
}
