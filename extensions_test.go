package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}

const goodManifest = `{"id":"demo-ext","name":"Demo","version":"1.2.3","matches":["*://*.example.com/*"],"js":["content.js"],"css":["style.css"]}`

func goodZip(t *testing.T) []byte {
	return makeZip(t, map[string]string{
		"manifest.json": goodManifest,
		"content.js":    "window.__demo = (window.__demo||0)+1;",
		"style.css":     "body{outline:1px solid red}",
	})
}

func TestInstallEnableDisableRemove(t *testing.T) {
	m := newExtensionManager(t.TempDir())
	m.Load()

	info, err := m.InstallZip(goodZip(t))
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "demo-ext" || info.Enabled {
		t.Fatalf("unexpected info %+v", info)
	}
	if _, err := os.Stat(m.installedPath("demo-ext")); err == nil {
		t.Fatal("installed dir must not exist before enabling")
	}

	if err := m.Enable("demo-ext"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.installedPath("demo-ext"), "content.js")); err != nil {
		t.Fatal("enable must extract the zip:", err)
	}
	if got := m.PayloadsFor("https://www.example.com/page"); len(got) != 1 {
		t.Fatalf("want 1 payload, got %d", len(got))
	} else if !strings.Contains(got[0], "window.__demo") || !strings.Contains(got[0], `"demo-ext"`) {
		t.Fatal("payload missing code or id")
	}
	if got := m.PayloadsFor("https://other.org/"); len(got) != 0 {
		t.Fatal("must not match other.org")
	}
	if got := m.PayloadsFor("sparrow://settings"); len(got) != 0 {
		t.Fatal("must not match internal pages")
	}

	// persisted across restarts
	m2 := newExtensionManager(m.root)
	m2.Load()
	if len(m2.PayloadsFor("https://example.com/")) != 1 {
		t.Fatal("enabled state must survive restart")
	}

	if err := m.Disable("demo-ext"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.installedPath("demo-ext")); err == nil {
		t.Fatal("disable must delete the extracted folder")
	}
	if _, err := os.Stat(m.packagePath("demo-ext")); err != nil {
		t.Fatal("disable must keep the zip")
	}
	if len(m.PayloadsFor("https://example.com/")) != 0 {
		t.Fatal("disabled extension must not inject")
	}
	if l := m.List(); len(l) != 1 || l[0].Enabled {
		t.Fatalf("bad list %+v", l)
	}
	if err := m.Remove("demo-ext"); err != nil {
		t.Fatal(err)
	}
	if len(m.List()) != 0 {
		t.Fatal("remove must delete the package")
	}
}

func TestTopLevelFolderZip(t *testing.T) {
	m := newExtensionManager(t.TempDir())
	m.Load()
	data := makeZip(t, map[string]string{
		"my-ext/manifest.json": goodManifest,
		"my-ext/content.js":    "1;",
		"my-ext/style.css":     "a{}",
	})
	if _, err := m.InstallZip(data); err != nil {
		t.Fatal(err)
	}
	if err := m.Enable("demo-ext"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.installedPath("demo-ext"), "content.js")); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsBadPackages(t *testing.T) {
	m := newExtensionManager(t.TempDir())
	m.Load()
	cases := map[string][]byte{
		"not a zip":      []byte("hello"),
		"no manifest":    makeZip(t, map[string]string{"a.js": "1"}),
		"bad id":         makeZip(t, map[string]string{"manifest.json": `{"id":"../x","name":"x","js":["a.js"]}`, "a.js": "1"}),
		"missing file":   makeZip(t, map[string]string{"manifest.json": `{"id":"x","name":"x","js":["nope.js"]}`}),
		"path traversal": makeZip(t, map[string]string{"manifest.json": `{"id":"x","name":"x","js":["../a.js"]}`}),
		"bad run_at":     makeZip(t, map[string]string{"manifest.json": `{"id":"x","name":"x","run_at":"never","js":["a.js"]}`, "a.js": "1"}),
		"bad match":      makeZip(t, map[string]string{"manifest.json": `{"id":"x","name":"x","matches":["nonsense"],"js":["a.js"]}`, "a.js": "1"}),
		"no js or css":   makeZip(t, map[string]string{"manifest.json": `{"id":"x","name":"x"}`}),
	}
	for name, data := range cases {
		if _, err := m.InstallZip(data); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestZipSlipBlocked(t *testing.T) {
	m := newExtensionManager(t.TempDir())
	m.Load()
	data := makeZip(t, map[string]string{
		"manifest.json":   `{"id":"evil","name":"evil","js":["a.js"]}`,
		"a.js":            "1",
		"../../escape.js": "pwned",
	})
	if _, err := m.InstallZip(data); err != nil {
		t.Fatal(err)
	}
	if err := m.Enable("evil"); err == nil {
		t.Fatal("zip-slip entry must make enable fail")
	}
	if _, err := os.Stat(filepath.Join(m.root, "..", "escape.js")); err == nil {
		t.Fatal("file escaped the extension folder")
	}
}

func TestMatchPatterns(t *testing.T) {
	cases := []struct {
		pattern, url string
		want         bool
	}{
		{"<all_urls>", "https://a.com/x", true},
		{"<all_urls>", "file:///C:/a.html", true},
		{"<all_urls>", "data:text/html,hi", false},
		{"<all_urls>", "sparrow://home", false},
		{"*://*.youtube.com/*", "https://www.youtube.com/watch?v=1", true},
		{"*://*.youtube.com/*", "https://youtube.com/", true},
		{"*://*.youtube.com/*", "https://notyoutube.com/", false},
		{"https://example.com/a/*", "https://example.com/a/b", true},
		{"https://example.com/a/*", "http://example.com/a/b", false},
		{"*://localhost/*", "http://localhost:3000/x", true},
	}
	for _, c := range cases {
		re, err := compileMatchPattern(c.pattern)
		if err != nil {
			t.Fatalf("%s: %v", c.pattern, err)
		}
		if got := re.MatchString(c.url); got != c.want {
			t.Errorf("%s vs %s = %v, want %v", c.pattern, c.url, got, c.want)
		}
	}
}

func TestTokenGate(t *testing.T) {
	if !strings.Contains(extListJSON("wrong"), `"ok":false`) {
		t.Fatal("wrong token must be rejected")
	}
	if !strings.Contains(extListJSON(extUIToken), `"ok":true`) {
		t.Fatal("right token must work")
	}
}
