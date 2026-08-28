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

// RulesSchemaVersion is the current rule-snapshot JSON schema. Unversioned blobs are rejected.
const RulesSchemaVersion = 1

// SnapshotRule describes one rule in a snapshot. The host Compiler maps plugin to an Inspector; the kernel does not match.
type SnapshotRule struct {
	ID      string `json:"id"`
	Plugin  string `json:"plugin,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Type    string `json:"type,omitempty"`
	Action  string `json:"action,omitempty"`
}

// RuleSnapshot is persistable rule config (AC-8 / AC-13). The file is a carrier, not a SQLite table.
type RuleSnapshot struct {
	SchemaVersion int            `json:"schema_version"`
	Rules         []SnapshotRule `json:"rules"`
}

// Compiler turns a snapshot into Inspectors. The host injects it (e.g. keyword plugin) so the kernel does not import plugin packages.
type Compiler func(RuleSnapshot) ([]Inspector, error)

// Ruleset is an atomically replaceable in-process rule set. The hot path reads the current snapshot per request, not inside the chunk loop.
type Ruleset struct {
	mu          sync.RWMutex
	compile     Compiler
	inspectors  []Inspector
	snap        RuleSnapshot
	fingerprint string
}

// NewRuleset creates an empty set. Inspectors exist only after the first Load / LoadFile.
func NewRuleset(compile Compiler) *Ruleset {
	return &Ruleset{compile: compile}
}

// Inspectors returns a copy of the current plugin slice for one request.
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

// Snapshot returns a copy of the currently loaded snapshot.
func (s *Ruleset) Snapshot() RuleSnapshot {
	if s == nil {
		return RuleSnapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSnapshot(s.snap)
}

// Load compiles and replaces rules. On failure the previous rules stay.
func (s *Ruleset) Load(snap RuleSnapshot) error {
	return s.apply(snap, "")
}

// LoadFile loads from a JSON file. On failure the previous rules stay.
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

// ReloadIfChanged compiles only when file bytes differ from the last successful load. Missing file returns that error.
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

// ParseSnapshot parses a rule-snapshot JSON.
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

// SeedSnapshot builds the first snapshot from env seeds when the rules file does not exist yet.
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

// WriteSnapshotFile writes a snapshot as JSON for ops or the control plane.
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

// WatchRulesFile polls the rules file. Compile failure keeps the old rules. interval<=0 returns immediately.
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
