package anilist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var graphqlURL = "https://graphql.anilist.co"

// Media represents the AniList Media fields we use (manga search result).
type Media struct {
	ID          int64       `json:"id"`
	SiteURL     string      `json:"siteUrl"`
	Title       *Title      `json:"title"`
	CoverImage  *CoverImage `json:"coverImage"`
	BannerImage string      `json:"bannerImage"`
}

type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
}

type CoverImage struct {
	Large string `json:"large"`
}

// graphqlResponse matches the AniList API response for our query.
type graphqlResponse struct {
	Data *struct {
		Media *Media `json:"Media"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"errors"`
}

// SearchManga searches AniList for a manga by title and returns the first match, or nil if not found.
func SearchManga(title string) (*Media, error) {
	cleanTitle := cleanTitleForSearch(title)
	if cleanTitle == "" {
		return nil, nil
	}

	query := `query ($search: String) {
  Media(search: $search, type: MANGA) {
    id
    title { romaji english }
    coverImage { large }
    bannerImage
    siteUrl
  }
}`
	body := map[string]interface{}{
		"query":     query,
		"variables": map[string]string{"search": cleanTitle},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, graphqlURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var out graphqlResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil && anilistNotFound(out) {
			return nil, nil
		}
		log.Printf("AniList API error: %s", resp.Status)
		return nil, fmt.Errorf("anilist API returned %s", resp.Status)
	}

	var out graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		log.Printf("AniList API errors: %v", out.Errors)
		return nil, nil
	}
	if out.Data == nil || out.Data.Media == nil {
		return nil, nil
	}
	return out.Data.Media, nil
}

func anilistNotFound(out graphqlResponse) bool {
	for _, e := range out.Errors {
		if e.Status == http.StatusNotFound || strings.EqualFold(strings.Trim(e.Message, ". "), "not found") {
			return true
		}
	}
	return false
}

var nonWordRegexp = regexp.MustCompile(`[^\w\s-]`)

// cleanTitleForSearch strips non-word characters for better search results.
func cleanTitleForSearch(title string) string {
	s := nonWordRegexp.ReplaceAllString(title, "")
	s = strings.TrimSpace(s)
	return s
}
