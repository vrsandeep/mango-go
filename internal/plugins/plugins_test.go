package plugins

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// setupMockWebtoonsServer creates a mock HTTP server simulating webtoons.com API
func setupMockWebtoonsServer() *httptest.Server {
	mux := http.NewServeMux()
	var serverURL string

	// Mock search endpoint - webtoons search API response format
	mux.HandleFunc("/en/search/immediate", func(w http.ResponseWriter, r *http.Request) {
		// Check query parameter
		query := r.URL.Query().Get("keyword")

		w.Header().Set("Content-Type", "application/json")

		// Mock response for "unordinary" search
		if query == "unordinary" || query == "unord" || query == "uno" {
			w.Write([]byte(`{
				"result": {
					"total": 1,
					"searchedList": [
						{
							"titleNo": 6795,
							"title": "unOrdinary",
							"authorNameList": ["uru-chan"],
							"representGenre": "Drama",
							"thumbnailImage2": "/thumbnail/icon_webtoon/6795/thumbnail_icon_webtoon_6795.jpg",
							"thumbnailMobile": "/thumbnail/icon_webtoon/6795/thumbnail_icon_webtoon_6795.jpg"
						}
					]
				}
			}`))
			return
		}

		// Default empty result
		w.Write([]byte(`{"result": {"total": 0, "searchedList": []}}`))
	})

	// Mock episode list redirect
	mux.HandleFunc("/episodeList", func(w http.ResponseWriter, r *http.Request) {
		titleNo := r.URL.Query().Get("titleNo")
		if titleNo == "6795" {
			// Redirect to mobile URL (use absolute URL from server)
			redirectURL := serverURL + "/en/drama/unordinary/list?title_no=" + titleNo
			w.Header().Set("Location", redirectURL)
			w.WriteHeader(http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	// Mock mobile episode list page
	mux.HandleFunc("/en/drama/unordinary/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<head><title>unOrdinary</title></head>
				<body>
					<h2 class="subj">unOrdinary</h2>
					<ul id="_episodeList">
						<li id="episode_1">
							<a href="/en/drama/unordinary/episode-1/viewer?episode_no=1">
								<span class="ellipsis">Episode 1</span>
								<span class="col num">#1</span>
								<span class="date">May 24, 2016</span>
							</a>
						</li>
						<li id="episode_2">
							<a href="/en/drama/unordinary/episode-2/viewer?episode_no=2">
								<span class="ellipsis">Episode 2</span>
								<span class="col num">#2</span>
								<span class="date">May 31, 2016</span>
							</a>
						</li>
					</ul>
				</body>
			</html>
		`))
	})

	// Mock viewer page
	mux.HandleFunc("/en/drama/unordinary/episode-1/viewer", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<body>
					<div class="subj_info">
						<a href="/en/drama/unordinary/list">unOrdinary</a>
						<span class="subj_episode">Episode 1</span>
					</div>
					<div id="_imageList">
						<img data-url="https://webtoon-phinf.pstatic.net/image1.jpg" />
						<img data-url="https://webtoon-phinf.pstatic.net/image2.jpg" />
						<img data-url="https://webtoon-phinf.pstatic.net/image3.jpg" />
					</div>
				</body>
			</html>
		`))
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	return server
}

func TestWebtoonsPlugin(t *testing.T) {
	// Setup test app
	app := testutil.SetupTestApp(t)

	// Setup mock server
	server := setupMockWebtoonsServer()
	defer server.Close()

	// Create temporary plugin directory
	pluginDir := t.TempDir()
	webtoonsDir := filepath.Join(pluginDir, "webtoons")
	os.MkdirAll(webtoonsDir, 0755)

	// Write plugin.json
	manifestJSON := `{
		"id": "webtoons",
		"name": "Webtoons",
		"version": "1.0.0",
		"description": "Download webtoons from webtoons.com",
		"author": "Test",
		"license": "MIT",
		"api_version": "1.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"capabilities": {
			"search": true,
			"chapters": true,
			"download": true
		},
		"config": {
			"base_url": {
				"type": "string",
				"default": "` + server.URL + `"
			},
			"mobile_url": {
				"type": "string",
				"default": "` + server.URL + `"
			}
		}
	}`
	os.WriteFile(filepath.Join(webtoonsDir, "plugin.json"), []byte(manifestJSON), 0644)

	// Write plugin code with configurable base URL
	pluginJS := `
const BASE_URL = mango.config.base_url || "https://www.webtoons.com";
const MOBILE_URL = mango.config.mobile_url || "https://m.webtoons.com";
const SEARCH_URL = BASE_URL + "/en/search/immediate?keyword=";
const SEARCH_PARAMS = "&q_enc=UTF-8&st=1&r_format=json&r_enc=UTF-8";
const THUMBNAIL_URL = "https://webtoon-phinf.pstatic.net";
const LIST_ENDPOINT = "/episodeList?titleNo=";

exports.getInfo = () => ({
  id: "webtoons",
  name: "Webtoons",
  version: "1.0.0"
});

exports.search = async (query, mango) => {
  mango.log.info("Searching Webtoons for: " + query);

  try {
    const searchUrl = SEARCH_URL + encodeURIComponent(query) + SEARCH_PARAMS;
    const headers = { 'Referer': BASE_URL + "/" };

    const response = await mango.http.get(searchUrl, { headers: headers });

    if (response.status !== 200) {
      throw new Error("Search failed: " + response.statusText);
    }

    const search = response.data;

    if (!search.result || search.result.total === 0) {
      mango.log.info("No results found");
      return [];
    }

    const searchedItems = search.result.searchedList || [];
    const results = searchedItems
      .filter(item => item.titleNo != null)
      .map(item => ({
        title: item.title || "Untitled",
        cover_url: item.thumbnailImage2 || item.thumbnailMobile
          ? THUMBNAIL_URL + (item.thumbnailImage2 || item.thumbnailMobile)
          : "",
        identifier: String(item.titleNo)
      }));

    mango.log.info("Found " + results.length + " results");
    return results;

  } catch (error) {
    mango.log.error("Search failed: " + error.message);
    throw new Error("Failed to search: " + error.message);
  }
};

exports.getChapters = async (seriesId, mango) => {
  mango.log.info("Fetching chapters for series: " + seriesId);

  try {
    const listUrl = BASE_URL + LIST_ENDPOINT + seriesId;
    const response = await mango.http.get(listUrl);

    let urlLocation = null;
    if (response.headers && response.headers.location) {
      urlLocation = response.headers.location;
    }

    if (!urlLocation) {
      throw new Error("Could not get webtoon page redirect");
    }

    const mobileUrl = MOBILE_URL + urlLocation;
    const mobileResponse = await mango.http.get(mobileUrl, {
      headers: { 'referer': MOBILE_URL }
    });

    if (mobileResponse.status !== 200) {
      throw new Error("Failed to fetch mobile page: " + mobileResponse.statusText);
    }

    const html = mobileResponse.text();

    const chapters = [];
    const titleMatch = html.match(/<h2[^>]*class="subj"[^>]*>([^<]+)</);
    const mangaTitle = titleMatch ? titleMatch[1].trim() : "Unknown";

    const episodeListRegex = /<li[^>]*id="[^"]*episode[^"]*"[^>]*>([\s\S]*?)<\/li>/g;
    let match;
    let episodeIndex = 0;

    while ((match = episodeListRegex.exec(html)) !== null) {
      episodeIndex++;
      const episodeHtml = match[1];

      const linkMatch = episodeHtml.match(/<a[^>]+href="([^"]+)"/);
      if (!linkMatch) continue;

      const url = linkMatch[1];
      const chapterIdMatch = url.match(/episode-(\d+)/);
      if (!chapterIdMatch) continue;

      const chapterSlug = chapterIdMatch[1];

      const titleMatch = episodeHtml.match(/<span[^>]*class="ellipsis"[^>]*>([^<]+)</);
      const chapterTitle = titleMatch ? titleMatch[1].trim() : "Episode " + episodeIndex;

      const numMatch = episodeHtml.match(/<span[^>]*class="col num"[^>]*>([^<]+)</);
      let chapterNum = "0";
      if (numMatch) {
        chapterNum = numMatch[1].replace("#", "").trim();
      }

      const dateMatch = episodeHtml.match(/<span[^>]*class="date"[^>]*>([^<]+)</);
      let publishedAt = new Date().toISOString();
      if (dateMatch) {
        const dateStr = dateMatch[1].replace("UP", "").trim();
        const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
        const parts = dateStr.split(" ");
        if (parts.length >= 3) {
          const month = months.indexOf(parts[0]);
          const day = parseInt(parts[1].replace(",", ""));
          const year = parseInt(parts[2]);
          if (month !== -1 && !isNaN(day) && !isNaN(year)) {
            publishedAt = new Date(year, month, day).toISOString();
          }
        }
      }

      const chapterId = "id" + seriesId + "ch" + chapterSlug + "num" + chapterNum;

      chapters.push({
        identifier: chapterId,
        title: chapterTitle,
        volume: "",
        chapter: chapterNum,
        pages: 0,
        language: "en",
        group_id: "",
        published_at: publishedAt
      });
    }

    chapters.reverse();
    chapters.sort((a, b) => {
      const aNum = parseFloat(a.chapter) || 0;
      const bNum = parseFloat(b.chapter) || 0;
      return aNum - bNum;
    });

    mango.log.info("Found " + chapters.length + " chapters");
    return chapters;

  } catch (error) {
    mango.log.error("GetChapters failed: " + error.message);
    throw new Error("Failed to get chapters: " + error.message);
  }
};

exports.getPageURLs = async (chapterId, mango) => {
  mango.log.info("Fetching page URLs for chapter: " + chapterId);

  try {
    const idMatch = chapterId.match(/id(\d+)ch(.+)num(.+)/);
    if (!idMatch) {
      throw new Error("Invalid chapter ID format");
    }

    const mangaID = idMatch[1];
    const chapterSlug = idMatch[2];
    const chapterNum = idMatch[3];

    const listUrl = BASE_URL + LIST_ENDPOINT + mangaID;
    const listResponse = await mango.http.get(listUrl);

    let urlLocation = null;
    if (listResponse.headers && listResponse.headers.location) {
      urlLocation = listResponse.headers.location;
    }

    if (!urlLocation) {
      throw new Error("Could not get webtoon chapter list");
    }

    const viewerUrl = BASE_URL + urlLocation.replace(/list/, chapterSlug + "/viewer") + "&episode_no=" + chapterNum;

    let finalUrl = viewerUrl;
    let attempts = 0;
    while (attempts < 5) {
      const resp = await mango.http.get(finalUrl);

      if (resp.status === 301 || resp.status === 302) {
        if (resp.headers && resp.headers.location) {
          finalUrl = BASE_URL + resp.headers.location;
          attempts++;
          continue;
        }
      } else if (resp.status === 200) {
        const html = resp.text();

        const imageListRegex = /<img[^>]+data-url="([^"]+)"/g;
        const imageUrls = [];
        let imgMatch;

        while ((imgMatch = imageListRegex.exec(html)) !== null) {
          imageUrls.push(imgMatch[1]);
        }

        if (imageUrls.length === 0) {
          throw new Error("No images found in chapter");
        }

        mango.log.info("Found " + imageUrls.length + " pages");
        return imageUrls;
      } else {
        throw new Error("Failed to get chapter viewer: " + resp.statusText);
      }
    }

    throw new Error("Too many redirects");

  } catch (error) {
    mango.log.error("GetPageURLs failed: " + error.message);
    throw new Error("Failed to get page URLs: " + error.message);
  }
};
`
	os.WriteFile(filepath.Join(webtoonsDir, "index.js"), []byte(pluginJS), 0644)

	// Load the plugin
	err := LoadPlugin(app, webtoonsDir)
	if err != nil {
		t.Fatalf("Failed to load plugin: %v", err)
	}

	// Cleanup
	t.Cleanup(func() {
		providers.UnregisterAll()
	})

	// Get the adapter
	provider, ok := providers.Get("webtoons")
	if !ok {
		t.Fatal("Plugin not registered")
	}

	adapter := provider.(*PluginProviderAdapter)

	t.Run("GetInfo", func(t *testing.T) {
		info := adapter.GetInfo()
		if info.ID != "webtoons" {
			t.Errorf("Expected ID 'webtoons', got '%s'", info.ID)
		}
		if info.Name != "Webtoons" {
			t.Errorf("Expected Name 'Webtoons', got '%s'", info.Name)
		}
	})

	t.Run("Search", func(t *testing.T) {
		results, err := adapter.Search("unordinary")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Expected at least 1 search result, got 0")
		}
		if results[0].Title != "unOrdinary" {
			t.Errorf("Expected title 'unOrdinary', got '%s'", results[0].Title)
		}
		if results[0].Identifier != "6795" {
			t.Errorf("Expected identifier '6795', got '%s'", results[0].Identifier)
		}
	})

	t.Run("GetChapters", func(t *testing.T) {
		chapters, err := adapter.GetChapters("6795")
		if err != nil {
			t.Fatalf("GetChapters() failed: %v", err)
		}
		if len(chapters) != 2 {
			t.Fatalf("Expected 2 chapters, got %d", len(chapters))
		}
		if chapters[0].Chapter != "1" {
			t.Errorf("Expected first chapter to be '1', got '%s'", chapters[0].Chapter)
		}
	})

	t.Run("GetPageURLs", func(t *testing.T) {
		urls, err := adapter.GetPageURLs("id6795chepisode-1num1")
		if err != nil {
			t.Fatalf("GetPageURLs() failed: %v", err)
		}
		if len(urls) != 3 {
			t.Fatalf("Expected 3 page URLs, got %d", len(urls))
		}
		if urls[0] != "https://webtoon-phinf.pstatic.net/image1.jpg" {
			t.Errorf("Expected first URL to be 'image1.jpg', got '%s'", urls[0])
		}
	})
}

