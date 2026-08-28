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
		t.Fatal("cannot locate test file")
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

func TestAC6_NextDoesNotProxyChatCompletions(t *testing.T) {
	root := repoRoot(t)
	cfg := readRepoFile(t, "controlplane/next.config.ts")
	if strings.Contains(cfg, "chat/completions") {
		t.Fatal("next.config must not proxy chat completions")
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
			t.Errorf("Route Handler must not proxy the chat stream: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAC9_DefaultFailOpenInEnvAndREADME(t *testing.T) {
	env := readRepoFile(t, ".env.example")
	readme := readRepoFile(t, "README.md")
	if !strings.Contains(env, "fail_open") || !strings.Contains(env, "CORAGATE_FAIL_MODE") {
		t.Fatal(".env.example must document default fail_open")
	}
	if !strings.Contains(readme, "fail_open") {
		t.Fatal("README must document default fail_open")
	}
}

func TestAC10_BuiltInKeywordPluginExists(t *testing.T) {
	path := filepath.Join(repoRoot(t), "plugins/detect/keyword/keyword.go")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing built-in detect plugin: %v", err)
	}
}

func TestAC12_READMEShowsClientAndCluster(t *testing.T) {
	readme := readRepoFile(t, "README.md")
	for _, want := range []string{"hosts/client", "hosts/cluster", "127.0.0.1", "0.0.0.0", "fleet"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing %q; both topologies required", want)
		}
	}
	for _, bad := range []string{"唯一中心 URL", "the only company-wide URL"} {
		if strings.Contains(readme, bad) {
			t.Fatalf("README must not present %q as the topology", bad)
		}
	}
}

func TestADR0009_EnglishREADMEHasChineseTranslation(t *testing.T) {
	en := readRepoFile(t, "README.md")
	zh := readRepoFile(t, "README.zh-CN.md")
	if !strings.Contains(en, "README.zh-CN.md") {
		t.Fatal("English README should link to the Chinese translation")
	}
	if !strings.Contains(zh, "舰队") {
		t.Fatal("Chinese translation should keep the fleet narrative")
	}
}

func TestAC14_KernelHasNoSQLiteTableDDL(t *testing.T) {
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
			t.Errorf("kernel must not hard-code SQLite tables: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
