package accheck

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAC2_ControlPlaneHasNoLocalePathSegment(t *testing.T) {
	app := filepath.Join(repoRoot(t), "controlplane/app")
	err := filepath.WalkDir(app, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == "[locale]" || name == "en" || name == "zh-CN" || name == "zh" {
			t.Errorf("locale must not be a route segment: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot(t), "controlplane/middleware.ts")); err == nil {
		mw := readRepoFile(t, "controlplane/middleware.ts")
		if strings.Contains(mw, "localePrefix") || strings.Contains(mw, "createMiddleware") {
			t.Fatal("middleware must not add locale path prefixes")
		}
	}
}

func TestAC3_SetLocaleActionStaysOnRoot(t *testing.T) {
	src := readRepoFile(t, "controlplane/app/actions.ts")
	if !strings.Contains(src, "setLocaleAction") {
		t.Fatal("missing setLocaleAction")
	}
	if !strings.Contains(src, "revalidatePath") {
		t.Fatal("setLocaleAction must revalidate / after writing the cookie")
	}
	if strings.Contains(src, `redirect("/en"`) || strings.Contains(src, `redirect("/zh`) {
		t.Fatal("setLocaleAction must not redirect to a locale prefix")
	}
	if !strings.Contains(readRepoFile(t, "controlplane/i18n/locale.ts"), "coragate_locale") {
		t.Fatal("cookie name coragate_locale must exist")
	}
}

func TestAC4_MessageCatalogKeysMatch(t *testing.T) {
	en := flattenJSONKeys(t, "controlplane/messages/en.json")
	zh := flattenJSONKeys(t, "controlplane/messages/zh-CN.json")
	if len(en) == 0 {
		t.Fatal("empty en catalog")
	}
	for k := range en {
		if _, ok := zh[k]; !ok {
			t.Errorf("zh-CN missing key %s", k)
		}
	}
	for k := range zh {
		if _, ok := en[k]; !ok {
			t.Errorf("en missing key %s", k)
		}
	}
}

func TestAC8_ControlPlaneREADMEDocumentsCookieNotPath(t *testing.T) {
	en := readRepoFile(t, "controlplane/README.md")
	zh := readRepoFile(t, "controlplane/README.zh-CN.md")
	for _, doc := range []string{en, zh} {
		if !strings.Contains(doc, "coragate_locale") {
			t.Fatal("README must document coragate_locale cookie")
		}
	}
	if strings.Contains(en, "visit /en") || strings.Contains(en, "open /zh-CN") {
		t.Fatal("English README must not present path prefixes as the usage")
	}
}

func flattenJSONKeys(t *testing.T, rel string) map[string]struct{} {
	t.Helper()
	var raw any
	if err := json.Unmarshal([]byte(readRepoFile(t, rel)), &raw); err != nil {
		t.Fatal(err)
	}
	out := map[string]struct{}{}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		m, ok := v.(map[string]any)
		if !ok {
			if prefix != "" {
				out[prefix] = struct{}{}
			}
			return
		}
		for k, child := range m {
			next := k
			if prefix != "" {
				next = prefix + "." + k
			}
			walk(next, child)
		}
	}
	walk("", raw)
	return out
}
