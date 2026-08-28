package kernel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RulesSchemaVersion 是规则快照 JSON 的当前 schema。不识别版本的 blob 一律拒绝加载。
const RulesSchemaVersion = 1

// SnapshotRule 是快照里的一条规则描述。plugin 由宿主 Compiler 映射到具体插件，内核不做匹配。
type SnapshotRule struct {
	ID      string `json:"id"`
	Plugin  string `json:"plugin,omitempty"`
	Pattern string `json:"pattern"`
}

// RuleSnapshot 是可落盘的规则配置（AC-8 / AC-13）。文件只是载体，不是 SQLite 表。
type RuleSnapshot struct {
	SchemaVersion int            `json:"schema_version"`
	Rules         []SnapshotRule `json:"rules"`
}

// Compiler 把快照编成 Inspector。宿主注入（例如关键字插件），避免内核 import 具体插件包。
type Compiler func(RuleSnapshot) ([]Inspector, error)

// Ruleset 是进程内可原子替换的规则集。热路径每次请求读当前快照，不在 chunk 循环里换插件。
type Ruleset struct {
	mu          sync.RWMutex
	compile     Compiler
	inspectors  []Inspector
	snap        RuleSnapshot
	fingerprint string
}

// NewRuleset 创建空规则集；首次 Load / LoadFile 之后才有 Inspector。
func NewRuleset(compile Compiler) *Ruleset {
	return &Ruleset{compile: compile}
}

// Inspectors 返回当前插件切片副本，供单次请求使用。
func (s *Ruleset) Inspectors() []Inspector {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Inspector, len(s.inspectors))
	copy(out, s.inspectors)
	return out
}

// Snapshot 返回当前已加载快照的副本。
func (s *Ruleset) Snapshot() RuleSnapshot {
	if s == nil {
		return RuleSnapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSnapshot(s.snap)
}

// Load 编译并替换规则。失败时保留旧规则。
func (s *Ruleset) Load(snap RuleSnapshot) error {
	return s.apply(snap, "")
}

// LoadFile 从 JSON 文件加载。失败时保留旧规则。
func (s *Ruleset) LoadFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	snap, err := ParseSnapshot(b)
	if err != nil {
		return err
	}
	return s.apply(snap, fileFingerprint(b))
}

// ReloadIfChanged 文件内容与上次成功加载不同才编译。文件不存在时返回该错误。
func (s *Ruleset) ReloadIfChanged(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	fp := fileFingerprint(b)
	s.mu.RLock()
	same := s.fingerprint != "" && s.fingerprint == fp
	s.mu.RUnlock()
	if same {
		return false, nil
	}
	snap, err := ParseSnapshot(b)
	if err != nil {
		return false, err
	}
	if err := s.apply(snap, fp); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Ruleset) apply(snap RuleSnapshot, fingerprint string) error {
	if s == nil || s.compile == nil {
		return fmt.Errorf("规则集未配置 Compiler")
	}
	if err := validateSnapshot(snap); err != nil {
		return err
	}
	ins, err := s.compile(snap)
	if err != nil {
		return err
	}
	if ins == nil {
		ins = []Inspector{}
	}
	s.mu.Lock()
	s.inspectors = ins
	s.snap = cloneSnapshot(snap)
	if fingerprint != "" {
		s.fingerprint = fingerprint
	}
	s.mu.Unlock()
	return nil
}

// ParseSnapshot 解析规则快照 JSON。
func ParseSnapshot(b []byte) (RuleSnapshot, error) {
	var snap RuleSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return RuleSnapshot{}, fmt.Errorf("规则快照 JSON: %w", err)
	}
	if err := validateSnapshot(snap); err != nil {
		return RuleSnapshot{}, err
	}
	if snap.Rules == nil {
		snap.Rules = []SnapshotRule{}
	}
	return snap, nil
}

func validateSnapshot(snap RuleSnapshot) error {
	if snap.SchemaVersion != RulesSchemaVersion {
		return fmt.Errorf("不支持的规则快照 schema_version=%d，需要 %d", snap.SchemaVersion, RulesSchemaVersion)
	}
	return nil
}

// SeedSnapshot 在规则文件尚不存在时，用环境变量种子生成第一份快照。
func SeedSnapshot(id, pattern string) RuleSnapshot {
	return RuleSnapshot{
		SchemaVersion: RulesSchemaVersion,
		Rules: []SnapshotRule{{
			ID:      id,
			Plugin:  "keyword",
			Pattern: pattern,
		}},
	}
}

// WriteSnapshotFile 把快照写成 JSON 文件，供运维或控制面编辑。
func WriteSnapshotFile(path string, snap RuleSnapshot) error {
	if snap.SchemaVersion == 0 {
		snap.SchemaVersion = RulesSchemaVersion
	}
	if err := validateSnapshot(snap); err != nil {
		return err
	}
	if snap.Rules == nil {
		snap.Rules = []SnapshotRule{}
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// WatchRulesFile 短轮询规则文件；编译失败保留旧规则。interval<=0 则立即返回。
func WatchRulesFile(ctx context.Context, rs *Ruleset, path string, interval time.Duration, logf func(string, ...any)) {
	if rs == nil || interval <= 0 || path == "" {
		return
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := rs.ReloadIfChanged(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				logf("coragate rules reload: %v", err)
				continue
			}
			if changed {
				logf("coragate rules reloaded path=%s schema_version=%d count=%d", path, rs.Snapshot().SchemaVersion, len(rs.Snapshot().Rules))
			}
		}
	}
}

func fileFingerprint(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func cloneSnapshot(snap RuleSnapshot) RuleSnapshot {
	out := snap
	if snap.Rules != nil {
		out.Rules = make([]SnapshotRule, len(snap.Rules))
		copy(out.Rules, snap.Rules)
	}
	return out
}
