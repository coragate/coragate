package kernel

import (
	"log"
	"net/http"
)

// ListenAndServe starts the dataplane HTTP server from Config.
func ListenAndServe(cfg Config) error {
	log.Printf("coragate listen=%s upstream=%s policy=%s fail=%s rules=%s", cfg.Listen, cfg.UpstreamBaseURL, cfg.policyMode(), parseFailMode(cfg.FailMode), cfg.RulesPath)
	return http.ListenAndServe(cfg.Listen, Handler(cfg))
}
