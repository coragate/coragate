package kernel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Version 为对外 SemVer。发布构建可用
// -ldflags "-X github.com/coragate/coragate/kernel.Version=x.y.z" 覆盖。
var Version = "0.1.0-dev"

// VersionInfo 是 CLI / 健康检查返回的版本面（AC-13）。
type VersionInfo struct {
	Version               string `json:"version"`
	ConfigSchemaVersion   int    `json:"config_schema_version"`
	RulesSchemaVersion    int    `json:"rules_schema_version"`
	EnvelopeSchemaVersion int    `json:"envelope_schema_version"`
}

// CurrentVersion 汇总二进制版本与各状态 schema。
func CurrentVersion() VersionInfo {
	return VersionInfo{
		Version:               Version,
		ConfigSchemaVersion:   ConfigSchemaVersion,
		RulesSchemaVersion:    RulesSchemaVersion,
		EnvelopeSchemaVersion: EnvelopeSchemaVersion,
	}
}

// WantVersion 表示进程应打印版本后退出，而不是启动监听。
func WantVersion(args []string) bool {
	for _, a := range args {
		if a == "--version" || a == "-version" {
			return true
		}
	}
	return false
}

// WriteVersion 把版本与 schema 写到 CLI（人类可读）。
func WriteVersion(w io.Writer) {
	v := CurrentVersion()
	fmt.Fprintf(w, "coragate %s\nconfig_schema_version=%d rules_schema_version=%d envelope_schema_version=%d\n",
		v.Version, v.ConfigSchemaVersion, v.RulesSchemaVersion, v.EnvelopeSchemaVersion)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(CurrentVersion())
}
