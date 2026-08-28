package kernel

import (
	"encoding/json"
	"net/http"
)

type reloadResponse struct {
	OK            bool `json:"ok"`
	SchemaVersion int  `json:"schema_version"`
	Rules         int  `json:"rules"`
}

func handleReload(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
		return
	}
	if cfg.Rules == nil || cfg.RulesPath == "" {
		http.Error(w, `{"error":{"message":"rules file not configured","type":"coragate_error"}}`, http.StatusServiceUnavailable)
		return
	}
	if err := cfg.Rules.LoadFile(cfg.RulesPath); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": err.Error(),
				"type":    "coragate_error",
			},
		})
		return
	}
	snap := cfg.Rules.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reloadResponse{
		OK:            true,
		SchemaVersion: snap.SchemaVersion,
		Rules:         len(snap.Rules),
	})
}
