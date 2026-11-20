package plugins

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// pluginIDPattern keeps an installed plugin in a single directory name.
var pluginIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// validatePluginID accepts a single path segment: letters, digits, '.', '_', and '-'.
// The id is used as the plugin directory name, so separators and ".." are rejected.
func validatePluginID(pluginID string) error {
	if !pluginIDPattern.MatchString(pluginID) {
		return fmt.Errorf("invalid plugin id %q", pluginID)
	}
	return nil
}

// validateEntryPoint rejects absolute paths, traversal, and non-script names.
func validateEntryPoint(entry string) error {
	if entry == "" || len(entry) > 128 || strings.Contains(entry, "\\") {
		return fmt.Errorf("invalid plugin entry point")
	}
	if path.Clean(entry) != entry || strings.HasPrefix(entry, "/") || strings.Contains(entry, "..") {
		return fmt.Errorf("invalid plugin entry point")
	}
	if !strings.HasSuffix(entry, ".js") {
		return fmt.Errorf("plugin entry point must be a .js file")
	}
	for _, part := range strings.Split(entry, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid plugin entry point")
		}
	}
	return nil
}

// joinWithin joins rel onto root and reports an error when the result leaves root.
func joinWithin(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes plugin directory")
	}
	root = filepath.Clean(root)
	full := filepath.Clean(filepath.Join(root, rel))
	relPath, err := filepath.Rel(root, full)
	if err != nil {
		return "", fmt.Errorf("path escapes plugin directory")
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes plugin directory")
	}
	return full, nil
}
