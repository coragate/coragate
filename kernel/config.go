package kernel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// HostKind selects the default listen address for a host. It does not change kernel behavior.
type HostKind int

const (
	HostCluster HostKind = iota
	HostClient
)

// Default listen addresses for the two hosts. The kernel does not assume cluster DNS; override with CORAGATE_LISTEN.
const (
	DefaultListenClient  = "127.0.0.1:8080"
	DefaultListenCluster = "0.0.0.0:8080"
)

// Config is the dataplane runtime configuration.
type Config struct {
	Listen          string
	UpstreamBaseURL string
	HTTPClient      *http.Client
	PolicyMode      string
	FailMode        string
	Inspectors      []Inspector
	Rules           *Ruleset
	RulesPath       string
	Auditor         *Auditor
	Plugins         []PluginInfo
}

// inspectors returns plugins for one request. When Rules is set, the snapshot wins so reload takes effect on the next request.
func (c Config) inspectors() []Inspector {
	if c.Rules != nil {
		return c.Rules.Inspectors()
	}
	return c.Inspectors
}

func (c Config) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	// Streaming responses must not set a client Timeout; it would cut the body mid-read.
	return &http.Client{Timeout: 0}
}

// LoadConfig reads environment variables. The host only supplies the default Listen.
func LoadConfig(host HostKind) Config {
	listen := os.Getenv("CORAGATE_LISTEN")
	if listen == "" {
		if host == HostClient {
			listen = DefaultListenClient
		} else {
			listen = DefaultListenCluster
		}
	}
	mode := os.Getenv("CORAGATE_POLICY_MODE")
	if mode == "" {
		mode = PolicyEnforce
	}
	return Config{
		Listen:          listen,
		UpstreamBaseURL: os.Getenv("CORAGATE_UPSTREAM_BASE_URL"),
		PolicyMode:      mode,
		FailMode:        parseFailMode(os.Getenv("CORAGATE_FAIL_MODE")),
	}
}

func (c Config) policyMode() string {
	if c.PolicyMode == PolicyObserve {
		return PolicyObserve
	}
	return PolicyEnforce
}

func parseFailMode(s string) string {
	if s == FailClosed {
		return FailClosed
	}
	return FailOpen
}

func (c Config) failClosed() bool {
	return parseFailMode(c.FailMode) == FailClosed
}

// ConfigSchemaVersion is the current runtime-config snapshot schema. Missing version is an opaque blob and is rejected.
const ConfigSchemaVersion = 1

// ConfigSnapshot is a versioned config shape (AC-13). Not a SQL table; secrets are omitted.
type ConfigSnapshot struct {
	SchemaVersion int    `json:"schema_version"`
	Listen        string `json:"listen,omitempty"`
	PolicyMode    string `json:"policy_mode,omitempty"`
	FailMode      string `json:"fail_mode,omitempty"`
	RulesPath     string `json:"rules_path,omitempty"`
}

// Snapshot exports non-secret config for health checks and upgrade detection.
func (c Config) Snapshot() ConfigSnapshot {
	return ConfigSnapshot{
		SchemaVersion: ConfigSchemaVersion,
		Listen:        c.Listen,
		PolicyMode:    c.policyMode(),
		FailMode:      parseFailMode(c.FailMode),
		RulesPath:     c.RulesPath,
	}
}

// ParseConfigSnapshot parses config JSON. Missing or unknown schema_version is rejected.
func ParseConfigSnapshot(b []byte) (ConfigSnapshot, error) {
	var snap ConfigSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return ConfigSnapshot{}, fmt.Errorf("配置快照 JSON: %w", err)
	}
	if snap.SchemaVersion != ConfigSchemaVersion {
		return ConfigSnapshot{}, fmt.Errorf("不支持的配置快照 schema_version=%d，需要 %d", snap.SchemaVersion, ConfigSchemaVersion)
	}
	return snap, nil
}
