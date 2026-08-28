package kernel

import (
	"encoding/json"
	"io"
	"net/http"
)

func writeBlocked(w http.ResponseWriter, ruleID string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(headerGatewayMark, "1")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"message": "blocked by rule " + ruleID,
			"type":    "coragate_policy",
			"code":    "content_filter",
		},
	})
}

func writeEngineClosed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(headerGatewayMark, "1")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"message": "detection engine unavailable",
			"type":    "coragate_engine",
			"code":    "engine_unavailable",
		},
	})
}

type inspectRequest struct {
	Text string `json:"text"`
}

type inspectResponse struct {
	Hit         bool   `json:"hit"`
	RuleID      string `json:"rule_id,omitempty"`
	EngineError string `json:"engine_error,omitempty"`
	EntityType  string `json:"entity_type,omitempty"`
	Action      string `json:"action,omitempty"`
}

func handleInspect(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"message":"read body","type":"coragate_error"}}`, http.StatusBadRequest)
		return
	}
	var req inspectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":{"message":"invalid json","type":"coragate_error"}}`, http.StatusBadRequest)
		return
	}
	hit := InspectInput(r.Context(), cfg.inspectors(), req.Text)
	w.Header().Set("Content-Type", "application/json")
	if hit.EngineError != "" {
		_ = json.NewEncoder(w).Encode(inspectResponse{EngineError: hit.EngineError})
		return
	}
	if hit.Hit {
		w.Header().Set(headerHitRule, hit.RuleID)
	}
	m := primaryMatch(hit)
	action := ""
	if hit.Hit {
		action = MatchAction(m)
	}
	_ = json.NewEncoder(w).Encode(inspectResponse{
		Hit:        hit.Hit,
		RuleID:     hit.RuleID,
		EntityType: m.EntityType,
		Action:     action,
	})
}
