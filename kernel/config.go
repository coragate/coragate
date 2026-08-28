package kernel

import (
	"net/http"
	"os"
)

// HostKind 区分两套宿主的默认监听，不改变内核行为。
type HostKind int

const (
	HostCluster HostKind = iota
	HostClient
)

// Config 为数据面运行配置。
type Config struct {
	Listen          string
	UpstreamBaseURL string
	HTTPClient      *http.Client
	PolicyMode      string
	Inspectors      []Inspector
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
			listen = "127.0.0.1:8080"
		} else {
			listen = "0.0.0.0:8080"
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
	}
}

func (c Config) policyMode() string {
	if c.PolicyMode == PolicyObserve {
		return PolicyObserve
	}
	return PolicyEnforce
}
