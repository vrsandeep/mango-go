package api

import (
	"path/filepath"
	"strings"

	"github.com/vrsandeep/mango-go/internal/models"
)

// relativeToLibrary returns full as a path under libraryPath.
// A path that escapes the library, including a sibling such as /manga2 when
// the library is /manga, is reduced to its base name.
func relativeToLibrary(libraryPath, full string) string {
	if full == "" {
		return ""
	}
	cleaned := filepath.Clean(full)
	if libraryPath == "" || !filepath.IsAbs(cleaned) {
		return filepath.ToSlash(cleaned)
	}
	rel, err := filepath.Rel(filepath.Clean(libraryPath), cleaned)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.Base(cleaned)
	}
	if rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (s *Server) libraryRelative(full string) string {
	if s.app == nil || s.app.Config() == nil {
		return full
	}
	return relativeToLibrary(s.app.Config().Library.Path, full)
}

func (s *Server) redactFolder(folder *models.Folder) {
	if folder == nil {
		return
	}
	folder.Path = s.libraryRelative(folder.Path)
}

func (s *Server) redactChapter(chapter *models.Chapter) {
	if chapter == nil {
		return
	}
	chapter.Path = s.libraryRelative(chapter.Path)
}
