package util

import "testing"

func TestNaturalSortLess(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"ch 2", "ch 10", true},
		{"chapter 10", "chapter 2", false},
		{"file1.jpg", "file10.jpg", true},
		{"file10.jpg", "file2.jpg", false},
		{"v1.2", "v1.10", true},
		{"item-1", "item-2", true},
		{"a", "b", true},
		{"b", "a", false},
		{"file", "file1", true},
		{"file1", "file", false},
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}
func TestNaturalSortLess_Equal(t *testing.T) {
	testCases := []struct {
		s1, s2 string
	}{
		{"chapter 1", "chapter 1"},
		{"file1.jpg", "file1.jpg"},
		{"v1.0", "v1.0"},
		{"item-1", "item-1"},
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result {
			t.Errorf("NaturalSortLess(%q, %q) = true; want false (equal case)", tc.s1, tc.s2)
		}
	}
}

func TestNaturalSortLess_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"file-1", "file-2", true},
		{"file_1", "file_10", true},
		{"file.1", "file.2", true},
		{"file@1", "file@10", true},
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}

func TestNaturalSortLess_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"File1", "file1", false},
		{"file1", "File1", false},
		{"Chapter2", "chapter2", false},
		{"chapter2", "Chapter2", false},
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}

func TestNaturalSortLess_ComplexCases(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.10", "v1.0.2", false},
		{"item-1a", "item-1b", true},
		{"item-1b", "item-1a", false},
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}
func TestNaturalSortLess_Unicode(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"café", "cafe", false},    // Unicode vs ASCII
		{"café", "café", false},    // Same Unicode
		{"café1", "café2", true},   // Unicode with numbers
		{"café10", "café2", false}, // Unicode with numbers
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}
func TestNaturalSortLess_Whitespace(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"file 1", "file 2", true},   // Space in strings
		{"file 10", "file 2", false}, // Space in strings
		{" file1", "file1", true},    // Leading space
		{"file1 ", "file1", false},   // Trailing space
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}
func TestNaturalSortLess_LongStrings(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"a very long string that goes on and on and on", "a very long string that goes on and on and on", false},
		{"a very long string that goes on and on and on 1", "a very long string that goes on and on and on 2", true},
		{"a very long string with numbers 10", "a very long string with numbers 2", false},
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}
func TestNaturalSortLess_Symbols(t *testing.T) {
	testCases := []struct {
		s1, s2   string
		expected bool
	}{
		{"file-1", "file-2", true},  // Hyphen
		{"file_1", "file_10", true}, // Underscore
		{"file.1", "file.2", true},  // Dot
		{"file@1", "file@10", true}, // At symbol
		{"file#1", "file#2", true},  // Hash symbol
	}
	for _, tc := range testCases {
		if result := NaturalSortLess(tc.s1, tc.s2); result != tc.expected {
			t.Errorf("NaturalSortLess(%q, %q) = %v; want %v", tc.s1, tc.s2, result, tc.expected)
		}
	}
}
