package util

import (
	"fmt"
	"slices"
	"testing"
)

func TestChapterSortSubstrings(t *testing.T) {
	expectedChapters := []string{"Vol.1 Ch.01", "Vol.1 Ch.02", "Vol.2 Ch. 2.5", "Ch. 3", "Ch.04"}

	receivedChapters := reverseSlice(expectedChapters)
	cs := NewChapterSorter(receivedChapters)

	slices.SortFunc(receivedChapters, func(a, b string) int {
		return cs.Compare(a, b)
	})
	for i, chapter := range receivedChapters {
		if chapter != expectedChapters[i] {
			t.Errorf("Expected chapter %d to be %q, got %q", i, expectedChapters[i], chapter)
		}
	}
	fmt.Println(
		"Expected sorted chapters:", expectedChapters,
		"\nActual sorted chapters:", receivedChapters,
	)

}

func TestChapterClubKeys(t *testing.T) {
	expectedChapters := []string{"Vol.1 Chapter 1", "Vol.1 Ch.02", "Vol.2 Ch. 2.5", "Ch. 3", "Ch.04"}

	// Reverse the order of chapters to test sorting
	receivedChapters := reverseSlice(expectedChapters)
	cs := NewChapterSorter(receivedChapters)

	slices.SortFunc(receivedChapters, func(a, b string) int {
		return cs.Compare(a, b)
	})
	for i, chapter := range receivedChapters {
		if chapter != expectedChapters[i] {
			t.Errorf("Expected chapter %d to be %q, got %q", i, expectedChapters[i], chapter)
		}
	}
}

func reverseSlice(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
