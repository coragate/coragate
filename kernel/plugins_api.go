package kernel

import (
	"encoding/json"
	"net/http"
)

func handlePlugins(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
		return
	}
	plugins := cfg.Plugins
	if plugins == nil {
		plugins = []PluginInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"plugins": plugins})
}
