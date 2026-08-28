package accheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAC6_Next不代理ChatCompletions(t *testing.T) {
	root := repoRoot(t)
	cfg := readRepoFile(t, "controlplane/next.config.ts")
	if strings.Contains(cfg, "chat/completions") {
		t.Fatal("next.config 不得出现 chat completions 代理")
	}
	cp := filepath.Join(root, "controlplane")
	err := filepath.WalkDir(cp, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() && (name == "node_modules" || name == ".next") {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		base := name
		if base != "route.ts" && base != "route.js" && base != "route.tsx" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "chat/completions") {
			t.Errorf("Route Handler 不得代理聊天流: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAC9_默认fail_open写在env与README(t *testing.T) {
	env := readRepoFile(t, ".env.example")
	readme := readRepoFile(t, "README.md")
	if !strings.Contains(env, "fail_open") || !strings.Contains(env, "CORAGATE_FAIL_MODE") {
		t.Fatal(".env.example 须写明默认 fail_open")
	}
	if !strings.Contains(readme, "fail_open") {
		t.Fatal("README 须写明默认 fail_open")
	}
}

func TestAC10_存在内置关键字插件包(t *testing.T) {
	path := filepath.Join(repoRoot(t), "plugins/detect/keyword/keyword.go")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("缺少内置检测插件: %v", err)
	}
}

func TestAC12_README同时给出本机与集群(t *testing.T) {
	readme := readRepoFile(t, "README.md")
	for _, want := range []string{"hosts/client", "hosts/cluster", "127.0.0.1", "0.0.0.0", "舰队"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README 缺少 %q，不能只给一种拓扑", want)
		}
	}
	if strings.Contains(readme, "唯一中心 URL") {
		t.Fatal("README 不得把唯一中心 URL 写成拓扑")
	}
}

func TestAC14_内核无SQLite表硬编码(t *testing.T) {
	root := filepath.Join(repoRoot(t), "kernel")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := strings.ToLower(string(b))
		if strings.Contains(s, "create table") || strings.Contains(s, "github.com/mattn/go-sqlite3") || strings.Contains(s, "modernc.org/sqlite") {
			t.Errorf("内核不得焊死 SQLite 表: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
