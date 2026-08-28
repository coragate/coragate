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

type inspectRequest struct {
	Text string `json:"text"`
}

type inspectResponse struct {
	Hit    bool   `json:"hit"`
	RuleID string `json:"rule_id,omitempty"`
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
	if hit.Hit {
		w.Header().Set(headerHitRule, hit.RuleID)
	}
	_ = json.NewEncoder(w).Encode(inspectResponse{Hit: hit.Hit, RuleID: hit.RuleID})
}
