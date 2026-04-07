package chapterfiles

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIsSupportedChapterFile_ArchiveExtensions(t *testing.T) {
	for _, ext := range []string{"book.cbz", "a.zip", "x.cbr", "y.rar", "z.7z", "q.cb7"} {
		if !IsSupportedChapterFile(ext) {
			t.Errorf("expected supported: %q", ext)
		}
	}
	for _, ext := range []string{"doc.pdf", "a.txt", "noext", "image.jpg"} {
		if IsSupportedChapterFile(ext) {
			t.Errorf("expected unsupported: %q", ext)
		}
	}
}

func TestInspectChapterFile_Unsupported(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "unknown.xyz")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := InspectChapterFile(context.Background(), p)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestGetChapterPage_Unsupported(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "unknown.xyz")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := GetChapterPage(context.Background(), p, 0)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}
