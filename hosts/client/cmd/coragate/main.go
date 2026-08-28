package main

import (
	"fmt"
	"os"
)

// 本机宿主：默认 127.0.0.1。代理逻辑见 SPEC-gateway-mvp T1。
func main() {
	fmt.Fprintln(os.Stderr, "coragate client host: 默认 127.0.0.1，尚未代理（见 T1）。规范：https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp")
	os.Exit(0)
}
