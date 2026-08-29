package run

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coragate/coragate/kernel"
	"github.com/coragate/coragate/plugins/detect/injection"
	"github.com/coragate/coragate/plugins/detect/keyword"
	"github.com/coragate/coragate/plugins/detect/pii"
	"github.com/coragate/coragate/plugins/detect/secret"
	"github.com/coragate/coragate/plugins/storage/file"
)

// Main wires built-in plugins and starts the dataplane. The kernel does not import concrete plugin packages.
func Main(host kernel.HostKind) {
	if kernel.WantVersion(os.Args[1:]) {
		kernel.WriteVersion(os.Stdout)
		return
	}
	cfg := kernel.LoadConfig(host)
	cfg.Plugins = pluginCatalog()
	rulesPath := envOr("CORAGATE_RULES_PATH", "data/rules.json")
	rs := kernel.NewRuleset(compileSnapshot)
	if err := loadRules(rs, rulesPath); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	cfg.Rules = rs
	cfg.RulesPath = rulesPath
	auditPath := envOr("CORAGATE_AUDIT_PATH", "data/audit.jsonl")
	st, err := file.New(auditPath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	cfg.Auditor = kernel.NewAuditor(st, 256)
	if poll := parsePoll(os.Getenv("CORAGATE_RULES_POLL")); poll > 0 {
		go kernel.WatchRulesFile(context.Background(), rs, rulesPath, poll, log.Printf)
	}
	if err := kernel.ListenAndServe(cfg); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func loadRules(rs *kernel.Ruleset, path string) error {
	if _, err := os.Stat(path); err == nil {
		return rs.LoadFile(path)
	}
	snap := kernel.SeedSnapshot(
		envOr("CORAGATE_DETECT_RULE_ID", "demo-keyword"),
		envOr("CORAGATE_DETECT_PATTERN", keyword.DefaultPattern),
	)
	if err := rs.Load(snap); err != nil {
		return err
	}
	if err := kernel.WriteSnapshotFile(path, snap); err != nil {
		log.Printf("未能写出规则快照 %s: %v（仍用环境变量种子运行）", path, err)
	}
	return nil
}

func pluginCatalog() []kernel.PluginInfo {
	return []kernel.PluginInfo{keyword.Info(), pii.Info(), injection.Info(), secret.Info()}
}

func pluginFactories() map[string]func(kernel.SnapshotRule) (kernel.Inspector, error) {
	return map[string]func(kernel.SnapshotRule) (kernel.Inspector, error){
		kernel.PluginKeyword:   keyword.Compile,
		kernel.PluginPII:       pii.Compile,
		kernel.PluginInjection: injection.Compile,
		kernel.PluginSecret:    secret.Compile,
	}
}

func compileSnapshot(snap kernel.RuleSnapshot) ([]kernel.Inspector, error) {
	factories := pluginFactories()
	out := make([]kernel.Inspector, 0, len(snap.Rules))
	for _, r := range snap.Rules {
		plugin := r.Plugin
		if plugin == "" {
			plugin = kernel.PluginKeyword
		}
		fn, ok := factories[plugin]
		if !ok {
			return nil, fmt.Errorf("未知检测插件 %q", plugin)
		}
		p, err := fn(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func parsePoll(s string) time.Duration {
	if s == "" {
		return time.Second
	}
	if s == "0" || s == "off" || s == "false" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("CORAGATE_RULES_POLL 无效 %q，回退 1s", s)
		return time.Second
	}
	return d
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
