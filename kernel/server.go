package kernel

import (
	"log"
	"net/http"
)

// ListenAndServe 按配置启动数据面 HTTP。
func ListenAndServe(cfg Config) error {
	log.Printf("coragate listen=%s upstream=%s policy=%s rules=%s", cfg.Listen, cfg.UpstreamBaseURL, cfg.policyMode(), cfg.RulesPath)
	return http.ListenAndServe(cfg.Listen, Handler(cfg))
}
