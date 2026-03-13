package store_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestGetFolderAnilist_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	folder, err := s.CreateFolder("/library/Some Manga", "Some Manga", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	_, err = s.GetFolderAnilist(folder.ID)
	if err != store.ErrAnilistCacheNotFound {
		t.Errorf("GetFolderAnilist: expected ErrAnilistCacheNotFound, got %v", err)
	}
}

func TestSetFolderAnilist_GetFolderAnilist_RoundTrip(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	folder, err := s.CreateFolder("/library/One Piece", "One Piece", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	err = s.SetFolderAnilist(folder.ID, 30013, "https://anilist.co/manga/30013", "https://s4.anilist.co/file/anilistcdn/media/manga/cover/large/bx30013-5Qz2hOuR9WFb.jpg", "One Piece", "One Piece")
	if err != nil {
		t.Fatalf("SetFolderAnilist: %v", err)
	}

	cached, err := s.GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatalf("GetFolderAnilist: %v", err)
	}
	if cached.FolderID != folder.ID {
		t.Errorf("FolderID: got %d, want %d", cached.FolderID, folder.ID)
	}
	if cached.AnilistID != 30013 {
		t.Errorf("AnilistID: got %d, want 30013", cached.AnilistID)
	}
	if cached.SiteURL != "https://anilist.co/manga/30013" {
		t.Errorf("SiteURL: got %q", cached.SiteURL)
	}
	if cached.TitleRomaji != "One Piece" {
		t.Errorf("TitleRomaji: got %q", cached.TitleRomaji)
	}
	if cached.TitleEnglish != "One Piece" {
		t.Errorf("TitleEnglish: got %q", cached.TitleEnglish)
	}
	if cached.CoverImageURL == "" {
		t.Error("CoverImageURL: expected non-empty")
	}
}

func TestSetFolderAnilist_Upsert(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	folder, err := s.CreateFolder("/library/Manga", "Manga", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	err = s.SetFolderAnilist(folder.ID, 1, "https://anilist.co/manga/1", "", "Romaji", "English")
	if err != nil {
		t.Fatalf("SetFolderAnilist (first): %v", err)
	}

	// Upsert with new data
	err = s.SetFolderAnilist(folder.ID, 2, "https://anilist.co/manga/2", "https://cover.example/2.jpg", "New Romaji", "New English")
	if err != nil {
		t.Fatalf("SetFolderAnilist (second): %v", err)
	}

	cached, err := s.GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatalf("GetFolderAnilist: %v", err)
	}
	if cached.AnilistID != 2 {
		t.Errorf("AnilistID: got %d, want 2 (upsert)", cached.AnilistID)
	}
	if cached.SiteURL != "https://anilist.co/manga/2" {
		t.Errorf("SiteURL: got %q", cached.SiteURL)
	}
	if cached.TitleRomaji != "New Romaji" {
		t.Errorf("TitleRomaji: got %q", cached.TitleRomaji)
	}
	if cached.CoverImageURL != "https://cover.example/2.jpg" {
		t.Errorf("CoverImageURL: got %q", cached.CoverImageURL)
	}
}

func TestSetFolderAnilist_OptionalFieldsEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	folder, err := s.CreateFolder("/library/Minimal", "Minimal", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	err = s.SetFolderAnilist(folder.ID, 99, "https://anilist.co/manga/99", "", "", "")
	if err != nil {
		t.Fatalf("SetFolderAnilist: %v", err)
	}

	cached, err := s.GetFolderAnilist(folder.ID)
	if err != nil {
		t.Fatalf("GetFolderAnilist: %v", err)
	}
	if cached.SiteURL != "https://anilist.co/manga/99" {
		t.Errorf("SiteURL: got %q", cached.SiteURL)
	}
	if cached.TitleRomaji != "" || cached.TitleEnglish != "" || cached.CoverImageURL != "" {
		t.Errorf("optional fields should be empty: romaji=%q english=%q cover=%q", cached.TitleRomaji, cached.TitleEnglish, cached.CoverImageURL)
	}
}
