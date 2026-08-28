package file

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/coragate/coragate/kernel"
)

// Store 默认文件适配器：一行一条 envelope JSON。不是内核数据模型。
type Store struct {
	path string
	mu   sync.Mutex
}

// New 打开（或创建）JSONL 审计文件。
func New(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return &Store{path: path}, nil
}

func (s *Store) Append(_ context.Context, env kernel.Envelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(env)
}

func (s *Store) List(_ context.Context, limit int) ([]kernel.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []kernel.Envelope
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var env kernel.Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			return out, err
		}
		out = append(out, env)
	}
	if err := sc.Err(); err != nil {
		return out, err
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}
