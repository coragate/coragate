package run

import (
	"log"
	"os"

	"github.com/coragate/coragate/kernel"
	"github.com/coragate/coragate/plugins/detect/keyword"
)

// Main 组装内置插件并启动数据面。内核本身不 import 具体插件包。
func Main(host kernel.HostKind) {
	cfg := kernel.LoadConfig(host)
	p, err := keyword.New(keyword.Rule{
		ID:      envOr("CORAGATE_DETECT_RULE_ID", "demo-keyword"),
		Pattern: envOr("CORAGATE_DETECT_PATTERN", keyword.DefaultPattern),
	})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	cfg.Inspectors = []kernel.Inspector{p}
	if err := kernel.ListenAndServe(cfg); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
