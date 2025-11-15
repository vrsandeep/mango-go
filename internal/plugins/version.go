package plugins

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// CompareVersions compares two version strings semantically.
// Returns:
// - -1 if v1 < v2
// - 0 if v1 == v2
// - 1 if v1 > v2
// - error if either version string is invalid
func CompareVersions(v1, v2 string) (int, error) {
	// Strip leading 'v' if present (common in version strings)
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	version1, err := semver.NewVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("invalid version %s: %w", v1, err)
	}

	version2, err := semver.NewVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("invalid version %s: %w", v2, err)
	}

	return version1.Compare(version2), nil
}

// IsNewerVersion checks if v2 is newer than v1.
// Returns true if v2 > v1, false otherwise.
func IsNewerVersion(v1, v2 string) (bool, error) {
	comparison, err := CompareVersions(v1, v2)
	if err != nil {
		return false, err
	}
	return comparison < 0, nil
}

// IsValidVersion checks if a version string is valid semantic version.
func IsValidVersion(version string) bool {
	// Strip leading 'v' if present
	version = strings.TrimPrefix(version, "v")
	_, err := semver.NewVersion(version)
	return err == nil
}

