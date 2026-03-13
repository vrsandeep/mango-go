package anilist_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/anilist"
)

func TestSearchManga_EmptyOrInvalidTitle_ReturnsNil(t *testing.T) {
	// SearchManga uses cleanTitleForSearch; empty or non-word titles return nil without calling the API.
	tests := []struct {
		name  string
		title string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"only special chars", "***"},
		{"only punctuation", "!@#..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			media, err := anilist.SearchManga(tt.title)
			if err != nil {
				t.Fatalf("SearchManga(%q): %v", tt.title, err)
			}
			if media != nil {
				t.Errorf("SearchManga(%q): expected nil, got %+v", tt.title, media)
			}
		})
	}
}
