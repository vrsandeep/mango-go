package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePluginID(t *testing.T) {
	if err := validatePluginID("test-plugin"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "..", "../etc", "a/b", `a\b`, ".hidden"} {
		if err := validatePluginID(id); err == nil {
			t.Fatalf("expected %q to be rejected", id)
		}
	}
}

func TestJoinWithinEntryPoint(t *testing.T) {
	root := t.TempDir()
	got, err := joinWithin(root, "index.js")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "index.js") {
		t.Fatalf("got %s", got)
	}
	if _, err := joinWithin(root, "../secret.js"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if err := validateEntryPoint("src/index.js"); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{"../index.js", "/tmp/index.js", "index.txt", `..\index.js`} {
		if err := validateEntryPoint(entry); err == nil {
			t.Fatalf("expected %q to be rejected", entry)
		}
	}
}

func TestJoinWithinRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	escapes := []string{
		"",
		"..",
		"../secret.js",
		"foo/../../secret.js",
		"/etc/passwd",
		filepath.Join(root, "outside.js"),
	}
	for _, rel := range escapes {
		if _, err := joinWithin(root, rel); err == nil {
			t.Fatalf("joinWithin(%q) succeeded; path escapes must be rejected", rel)
		}
	}

	// A cleaned nested path that stays inside the root is still valid.
	got, err := joinWithin(root, filepath.Join("src", "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != filepath.Join(root, "src") {
		t.Fatalf("nested path escaped: %s", got)
	}
}

func TestEntryPointAndManifestRejectTraversal(t *testing.T) {
	for _, entry := range []string{"", "./index.js", "src/../index.js", "index.js/../../etc/passwd", strings.Repeat("a", 126) + ".js"} {
		if err := validateEntryPoint(entry); err == nil {
			t.Fatalf("expected entry point %q to be rejected", entry)
		}
	}

	dir := t.TempDir()
	manifest := `{"id":"ok-plugin","api_version":"1.0","entry_point":"../outside.js"}`
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(dir); err == nil {
		t.Fatal("LoadManifest accepted an entry point outside the plugin directory")
	}
}
