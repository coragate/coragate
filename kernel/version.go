package kernel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Version is the public SemVer. Release builds may override it with
// -ldflags "-X github.com/coragate/coragate/kernel.Version=x.y.z".
var Version = "0.1.0-dev"

// VersionInfo is the version surface returned by the CLI and health check (AC-13).
type VersionInfo struct {
	Version               string `json:"version"`
	ConfigSchemaVersion   int    `json:"config_schema_version"`
	RulesSchemaVersion    int    `json:"rules_schema_version"`
	EnvelopeSchemaVersion int    `json:"envelope_schema_version"`
}

// CurrentVersion aggregates the binary version and schema versions.
func CurrentVersion() VersionInfo {
	return VersionInfo{
		Version:               Version,
		ConfigSchemaVersion:   ConfigSchemaVersion,
		RulesSchemaVersion:    RulesSchemaVersion,
		EnvelopeSchemaVersion: EnvelopeSchemaVersion,
	}
}

// WantVersion is true when the process should print the version and exit instead of listening.
func WantVersion(args []string) bool {
	for _, a := range args {
		if a == "--version" || a == "-version" {
			return true
		}
	}
	return false
}

// WriteVersion writes version and schema numbers to the CLI (human-readable).
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
