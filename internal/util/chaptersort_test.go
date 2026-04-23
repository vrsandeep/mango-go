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

// TestChapterSortKeyMergeFalsePositive documents a bug in hasKeyWithVariation: sanitized
// "School" is "school", and strings.Contains("school", "ch") is true, so a "School" + N
// token is merged with the "Ch." top key and can overwrite the real chapter number. That
// makes a lower chapter (2) sort after a higher one (10) when "School" is followed by
// a large number. Expected order: Ch. 2, then Ch. 10. Buggy order: Ch. 10, then "Ch. 2"
// (parsed as 99).
func TestChapterSortKeyMergeFalsePositive(t *testing.T) {
	ch2 := "Vol. 1 Ch. 2 School 99 Test"
	ch10 := "Vol. 1 Ch. 10 Other"
	expected := []string{ch2, ch10}

	titles := []string{ch10, ch2}
	cs := NewChapterSorter(titles)
	slices.SortFunc(titles, func(a, b string) int {
		return cs.Compare(a, b)
	})

	for i, want := range expected {
		if i >= len(titles) || titles[i] != want {
			t.Fatalf("ascending by chapter: want index %d = %q, got %v (this case fails until key-merge is fixed)", i, want, titles)
		}
	}
}

// TestChapterSortKeyMerge_SaikiShaped checks Saiki-style titles (PSI puns, "School", decimal
// chapters) sort by global Ch in realistic pairs: 2<71<83<167. Regression for substring
// key-merge; uses only four basenames, no large embedded corpus.
func TestChapterSortKeyMerge_SaikiShaped(t *testing.T) {
	const (
		ch2   = "Vol. 1 Ch. 2 The Ab-PSI-lute Worst!- Nendou Riki"
		ch71  = "Vol. 7 Ch. 71 PK Academy's School FePSIval! (1st Half)"
		ch83  = "Vol. 8 Ch. 83 Kaidou's PSIspicion (1st Half)"
		ch167 = "Vol. 16 Ch. 167 Espers Should ExcerPSIse Extreme Caution (1st Half)"
	)
	titles := []string{ch167, ch2, ch83, ch71}
	cs := NewChapterSorter(titles)
	for _, p := range [][2]string{{ch2, ch71}, {ch71, ch83}, {ch83, ch167}} {
		c := cs.Compare(p[0], p[1])
		if c == 0 {
			t.Errorf("Compare(%q, %q) = 0, want <0", p[0], p[1])
		} else if c > 0 {
			t.Errorf("Compare(%q, %q) = %d, want <0", p[0], p[1], c)
		}
	}
}
