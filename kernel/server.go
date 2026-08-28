package kernel

import (
	"log"
	"net/http"
	"os"
)

// ListenAndServe 按配置启动数据面 HTTP。
func ListenAndServe(cfg Config) error {
	log.Printf("coragate listen=%s upstream=%s", cfg.Listen, cfg.UpstreamBaseURL)
	return http.ListenAndServe(cfg.Listen, Handler(cfg))
}

// Run 是宿主入口：加载配置并阻塞监听。
func Run(host HostKind) {
	cfg := LoadConfig(host)
	if err := ListenAndServe(cfg); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
