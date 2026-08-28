package kernel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// HostKind 区分两套宿主的默认监听，不改变内核行为。
type HostKind int

const (
	HostCluster HostKind = iota
	HostClient
)

// 两套宿主的默认监听。内核本身不假设集群 DNS；可用 CORAGATE_LISTEN 覆盖。
const (
	DefaultListenClient  = "127.0.0.1:8080"
	DefaultListenCluster = "0.0.0.0:8080"
)

// Config 为数据面运行配置。
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
}

// inspectors 单次请求读取当前插件。Rules 存在时以快照为准，便于 reload 后新请求立刻换规则。
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
	// 流式响应不能设整体 Timeout，否则会在读完 body 前被掐断。
	return &http.Client{Timeout: 0}
}

// LoadConfig 从环境变量读取；宿主只决定 Listen 默认值。
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

// ConfigSchemaVersion 是运行配置快照的当前 schema。无版本字段视为不透明 blob，拒绝解析。
const ConfigSchemaVersion = 1

// ConfigSnapshot 是可识别版本的配置形状（AC-13）。不是某张 SQL 表；密钥不进快照。
type ConfigSnapshot struct {
	SchemaVersion int    `json:"schema_version"`
	Listen        string `json:"listen,omitempty"`
	PolicyMode    string `json:"policy_mode,omitempty"`
	FailMode      string `json:"fail_mode,omitempty"`
	RulesPath     string `json:"rules_path,omitempty"`
}

// Snapshot 导出当前非机密配置，供健康检查与升级识别。
func (c Config) Snapshot() ConfigSnapshot {
	return ConfigSnapshot{
		SchemaVersion: ConfigSchemaVersion,
		Listen:        c.Listen,
		PolicyMode:    c.policyMode(),
		FailMode:      parseFailMode(c.FailMode),
		RulesPath:     c.RulesPath,
	}
}

// ParseConfigSnapshot 解析配置 JSON；缺少或未知 schema_version 一律拒绝。
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
