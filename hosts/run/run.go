package run

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coragate/coragate/kernel"
	"github.com/coragate/coragate/plugins/detect/keyword"
	"github.com/coragate/coragate/plugins/storage/file"
)

// Main 组装内置插件并启动数据面。内核本身不 import 具体插件包。
func Main(host kernel.HostKind) {
	cfg := kernel.LoadConfig(host)
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

func compileSnapshot(snap kernel.RuleSnapshot) ([]kernel.Inspector, error) {
	out := make([]kernel.Inspector, 0, len(snap.Rules))
	for _, r := range snap.Rules {
		plugin := r.Plugin
		if plugin == "" {
			plugin = "keyword"
		}
		switch plugin {
		case "keyword":
			p, err := keyword.New(keyword.Rule{ID: r.ID, Pattern: r.Pattern})
			if err != nil {
				return nil, err
			}
			out = append(out, p)
		default:
			return nil, fmt.Errorf("未知检测插件 %q", plugin)
		}
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
