package weebcentral

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/vrsandeep/mango-go/internal/models"
)

// WeebCentralProvider implements the Provider interface for WeebCentral.
type WeebCentralProvider struct {
	client  *http.Client
	baseURL string
}

func New() *WeebCentralProvider {
	return &WeebCentralProvider{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://weebcentral.com",
	}
}

func (p *WeebCentralProvider) GetInfo() models.ProviderInfo {
	return models.ProviderInfo{
		ID:   "weebcentral",
		Name: "WeebCentral",
	}
}

func (p *WeebCentralProvider) Search(query string) ([]models.SearchResult, error) {
	searchURL := fmt.Sprintf("%s/search/simple?location=main", p.baseURL)
	form := url.Values{}
	form.Set("text", query)
	body := bytes.NewBufferString(form.Encode())

	req, err := http.NewRequest("POST", searchURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Trigger", "quick-search-input")
	req.Header.Set("HX-Trigger-Name", "text")
	req.Header.Set("HX-Target", "quick-search-result")
	req.Header.Set("HX-Current-URL", p.baseURL+"/")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []models.SearchResult
	doc.Find("#quick-search-result > div > a").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if !exists {
			return
		}
		title := strings.TrimSpace(s.Find(".flex-1").Text())
		var image string
		if src, ok := s.Find("source").Attr("srcset"); ok {
			image = src
		} else if src, ok := s.Find("img").Attr("src"); ok {
			image = src
		}
		// Extract manga id from link assuming the format contains '/series/{id}/'
		idPart := ""
		parts := strings.Split(link, "/series/")
		if len(parts) > 1 {
			subparts := strings.Split(parts[1], "/")
			idPart = subparts[0]
		}
		if idPart == "" {
			return
		}
		results = append(results, models.SearchResult{
			Title:      title,
			CoverURL:   image,
			Identifier: idPart,
		})
	})
	if len(results) == 0 {
		return nil, errors.New("no results found")
	}
	return results, nil
}

func (p *WeebCentralProvider) GetChapters(seriesIdentifier string) ([]models.ChapterResult, error) {
	chapterURL := fmt.Sprintf("%s/series/%s/full-chapter-list", p.baseURL, seriesIdentifier)
	req, err := http.NewRequest("GET", chapterURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "chapter-list")
	req.Header.Set("HX-Current-URL", fmt.Sprintf("%s/series/%s", p.baseURL, seriesIdentifier))
	req.Header.Set("Referer", fmt.Sprintf("%s/series/%s", p.baseURL, seriesIdentifier))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var chapters []models.ChapterResult
	chapterRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	doc.Find("div.flex.items-center").Each(func(i int, s *goquery.Selection) {
		a := s.Find("a")
		chapterLink, exists := a.Attr("href")
		if !exists {
			return
		}
		chapterTitle := strings.TrimSpace(a.Find("span.grow > span").First().Text())
		var chapterNumber string
		match := chapterRegex.FindStringSubmatch(chapterTitle)
		if len(match) > 1 {
			chapterNumber = cleanChapterNumber(match[1])
		} else {
			chapterNumber = ""
		}
		// Extract chapter id from the URL assuming format contains '/chapters/{id}'
		chapterId := ""
		parts := strings.Split(chapterLink, "/chapters/")
		if len(parts) > 1 {
			chapterId = parts[1]
		}
		// Try to extract published date from a <time> tag or data attribute
		var publishedAt time.Time
		if timeTag := s.Find("time"); timeTag.Length() > 0 {
			if datetime, exists := timeTag.Attr("datetime"); exists {
				parsed, err := time.Parse(time.RFC3339, datetime)
				if err == nil {
					publishedAt = parsed
				}
			}
		}
		chapters = append(chapters, models.ChapterResult{
			Identifier:  chapterId,
			Title:       chapterTitle,
			Chapter:     chapterNumber,
			PublishedAt: publishedAt, // Will be zero if not found
		})
	})
	if len(chapters) == 0 {
		return nil, errors.New("no chapters found")
	}
	// Reverse to ascending order
	for i, j := 0, len(chapters)-1; i < j; i, j = i+1, j-1 {
		chapters[i], chapters[j] = chapters[j], chapters[i]
	}
	// Assign index as volume if needed, or sort by chapter number
	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Chapter < chapters[j].Chapter
	})
	return chapters, nil
}

func (p *WeebCentralProvider) GetPageURLs(chapterIdentifier string) ([]string, error) {
	pageURL := fmt.Sprintf("%s/chapters/%s/images?is_prev=False&reading_style=long_strip", p.baseURL, chapterIdentifier)
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", fmt.Sprintf("%s/chapters/%s", p.baseURL, chapterIdentifier))
	req.Header.Set("Referer", fmt.Sprintf("%s/chapters/%s", p.baseURL, chapterIdentifier))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	var pages []string
	doc.Find("section.flex-1 img").Each(func(i int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok && src != "" {
			pages = append(pages, src)
		}
	})
	if len(pages) == 0 {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			if src, ok := s.Attr("src"); ok && src != "" {
				pages = append(pages, src)
			}
		})
	}
	if len(pages) == 0 {
		return nil, errors.New("no pages found")
	}
	return pages, nil
}

func cleanChapterNumber(chapterStr string) string {
	cleaned := strings.TrimLeft(chapterStr, "0")
	if cleaned == "" {
		return "0"
	}
	return cleaned
}
