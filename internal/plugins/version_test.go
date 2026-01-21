package plugins_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/plugins"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
		wantErr  bool
	}{
		{"Equal versions", "1.0.0", "1.0.0", 0, false},
		{"v1 less than v2", "1.0.0", "1.0.1", -1, false},
		{"v1 greater than v2", "1.0.1", "1.0.0", 1, false},
		{"Minor version difference", "1.0.0", "1.1.0", -1, false},
		{"Major version difference", "1.0.0", "2.0.0", -1, false},
		{"Patch version difference", "1.0.0", "1.0.1", -1, false},
		{"Pre-release vs release", "1.0.0-alpha", "1.0.0", -1, false},
		{"Build metadata", "1.0.0", "1.0.0+build", 0, false},
		{"Invalid version v1", "invalid", "1.0.0", 0, true},
		{"Invalid version v2", "1.0.0", "invalid", 0, true},
		{"Complex versions", "1.2.3-beta.1", "1.2.3-beta.2", -1, false},
		{"Version with leading v", "v1.0.0", "1.0.0", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := plugins.CompareVersions(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("CompareVersions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
		wantErr  bool
	}{
		{"v2 is newer", "1.0.0", "1.0.1", true, false},
		{"v2 is older", "1.0.1", "1.0.0", false, false},
		{"Same version", "1.0.0", "1.0.0", false, false},
		{"Major version update", "1.0.0", "2.0.0", true, false},
		{"Minor version update", "1.0.0", "1.1.0", true, false},
		{"Invalid version", "invalid", "1.0.0", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := plugins.IsNewerVersion(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsNewerVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("IsNewerVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"Valid version", "1.0.0", true},
		{"Valid with pre-release", "1.0.0-alpha", true},
		{"Valid with build", "1.0.0+build", true},
		{"Valid complex", "1.2.3-beta.1+build.123", true},
		{"Invalid empty", "", false},
		{"Invalid format", "not.a.version", false},
		{"Version with leading v", "v1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plugins.IsValidVersion(tt.version)
			if result != tt.expected {
				t.Errorf("IsValidVersion(%q) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}
