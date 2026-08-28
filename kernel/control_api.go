package kernel

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

func handleRules(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	switch r.Method {
	case http.MethodGet:
		handleRulesGet(cfg, w)
	case http.MethodPut:
		handleRulesPut(cfg, w, r)
	default:
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
	}
}

func handleRulesGet(cfg Config, w http.ResponseWriter) {
	snap := RuleSnapshot{SchemaVersion: RulesSchemaVersion, Rules: []SnapshotRule{}}
	if cfg.Rules != nil {
		snap = cfg.Rules.Snapshot()
		if snap.SchemaVersion == 0 {
			snap.SchemaVersion = RulesSchemaVersion
		}
		if snap.Rules == nil {
			snap.Rules = []SnapshotRule{}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func handleRulesPut(cfg Config, w http.ResponseWriter, r *http.Request) {
	if cfg.Rules == nil || cfg.RulesPath == "" {
		http.Error(w, `{"error":{"message":"rules file not configured","type":"coragate_error"}}`, http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"message":"read body","type":"coragate_error"}}`, http.StatusBadRequest)
		return
	}
	snap, err := ParseSnapshot(body)
	if err != nil {
		writeJSONMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := WriteSnapshotFile(cfg.RulesPath, snap); err != nil {
		writeJSONMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := cfg.Rules.LoadFile(cfg.RulesPath); err != nil {
		writeJSONMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg.Rules.Snapshot())
}

func handleAudit(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		n, err := strconv.Atoi(q)
		if err == nil && n > 0 {
			limit = n
		}
		if limit > 200 {
			limit = 200
		}
	}
	items, err := cfg.Auditor.List(r.Context(), limit)
	if err != nil {
		writeJSONMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []Envelope{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

func writeJSONMessage(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"message": msg,
			"type":    "coragate_error",
		},
	})
}
