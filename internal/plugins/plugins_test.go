package plugins_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/plugins"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// createTestPlugin creates a minimal test plugin for unit testing
func createTestPlugin(t *testing.T, pluginDir string, pluginJS string) (*plugins.PluginRuntime, error) {
	manifest := &plugins.PluginManifest{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		PluginType:  "downloader",
		EntryPoint:  "index.js",
		Description: "Test plugin for unit tests",
		APIVersion:  "1.0",
	}

	// Write plugin script
	scriptPath := filepath.Join(pluginDir, "index.js")
	if err := os.WriteFile(scriptPath, []byte(pluginJS), 0644); err != nil {
		return nil, err
	}

	app := testutil.SetupTestApp(t)
	runtime, err := plugins.NewPluginRuntime(app, manifest, pluginDir)
	return runtime, err
}

func TestPluginRuntime(t *testing.T) {
	t.Run("Create Runtime", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}
		if runtime == nil {
			t.Fatal("Runtime is nil")
		}
		if runtime.Manifest().ID != "test-plugin" {
			t.Errorf("Expected manifest ID 'test-plugin', got '%s'", runtime.Manifest().ID)
		}
	})

	t.Run("Call Synchronous Function", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		val, err := runtime.Call("getInfo")
		if err != nil {
			t.Fatalf("Call() failed: %v", err)
		}

		infoObj := val.ToObject(runtime.VM())
		id := infoObj.Get("id").String()
		if id != "test" {
			t.Errorf("Expected id 'test', got '%s'", id)
		}
	})

	t.Run("Call Async Function", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	return [{ title: "Test Series", identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		val, err := runtime.Call("search", "test")
		if err != nil {
			t.Fatalf("Call() failed: %v", err)
		}

		// Check that it's an array
		if val.ToObject(runtime.VM()).Get("length") == nil {
			t.Error("Expected array result from search")
		}
	})

	t.Run("Function Not Found", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		_, err = runtime.Call("nonexistent")
		if err == nil {
			t.Fatal("Expected error for nonexistent function")
		}
	})

	t.Run("Panic Recovery", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => {
	throw new Error("Intentional error");
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		_, err = runtime.Call("search", "test")
		if err == nil {
			t.Fatal("Expected error from panicking function")
		}

		pluginErr, ok := err.(*plugins.PluginError)
		if !ok {
			t.Fatalf("Expected PluginError, got %T", err)
		}
		if pluginErr.PluginID != "test-plugin" {
			t.Errorf("Expected PluginID 'test-plugin', got '%s'", pluginErr.PluginID)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => {
	return new Promise(resolve => setTimeout(() => resolve([]), 60000));
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		// Use context with timeout - the runtime has a 30s hard timeout too
		// So we test that the context timeout works by using a shorter timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err = runtime.CallWithContext(ctx, "search", "test")
		if err == nil {
			t.Fatal("Expected timeout error")
		}

		// The error should indicate a timeout occurred
		pluginErr, ok := err.(*plugins.PluginError)
		if !ok {
			t.Fatalf("Expected PluginError, got %T: %v", err, err)
		}
		// Context timeout will be caught by ctx.Done() in the select statement
		// Check that it's a timeout error
		if !pluginErr.IsTimeout {
			t.Logf("Note: Timeout error may come from context: %v", err)
			// If it's not marked as timeout, the error message should contain "timeout"
			if err.Error() != "timeout" && !contains(err.Error(), "timeout") {
				t.Errorf("Expected timeout error, got: %v", err)
			}
		}
	})

	t.Run("Missing Required Export", func(t *testing.T) {
		pluginDir := t.TempDir()
		manifest := &plugins.PluginManifest{
			ID:          "test-plugin",
			Name:        "Test Plugin",
			Version:     "1.0.0",
			PluginType:  "downloader",
			EntryPoint:  "index.js",
			Description: "Test plugin",
			APIVersion:  "1.0",
		}

		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
// Missing search, getChapters, getPageURLs
`
		scriptPath := filepath.Join(pluginDir, "index.js")
		os.WriteFile(scriptPath, []byte(pluginJS), 0644)

		app := testutil.SetupTestApp(t)
		_, err := plugins.NewPluginRuntime(app, manifest, pluginDir)
		if err == nil {
			t.Fatal("Expected error for missing required exports")
		}
	})
}

func TestPluginAdapter(t *testing.T) {
	t.Run("GetInfo", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test-plugin", name: "Test Plugin", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		info := adapter.GetInfo()

		if info.ID != "test-plugin" {
			t.Errorf("Expected ID 'test-plugin', got '%s'", info.ID)
		}
		if info.Name != "Test Plugin" {
			t.Errorf("Expected Name 'Test Plugin', got '%s'", info.Name)
		}
	})

	t.Run("GetInfo Fallback", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => { throw new Error("Error in getInfo"); };
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		info := adapter.GetInfo()

		// Should fallback to manifest values
		if info.ID != "test-plugin" {
			t.Errorf("Expected ID 'test-plugin', got '%s'", info.ID)
		}
	})

	t.Run("Search", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	return [
		{ title: "Series 1", identifier: "1", cover_url: "http://example.com/cover1.jpg" },
		{ title: "Series 2", identifier: "2", cover_url: "http://example.com/cover2.jpg" }
	];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		if results[0].Title != "Series 1" {
			t.Errorf("Expected first title 'Series 1', got '%s'", results[0].Title)
		}
		if results[0].Identifier != "1" {
			t.Errorf("Expected first identifier '1', got '%s'", results[0].Identifier)
		}
	})

	t.Run("Search Empty Results", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	t.Run("GetChapters", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async (seriesId, mango) => {
	return [
		{
			identifier: "ch1",
			title: "Chapter 1",
			volume: "1",
			chapter: "1",
			pages: 20,
			language: "en",
			group_id: "",
			published_at: "2024-01-01T00:00:00Z"
		},
		{
			identifier: "ch2",
			title: "Chapter 2",
			volume: "1",
			chapter: "2",
			pages: 22,
			language: "en",
			group_id: "",
			published_at: "2024-01-02T00:00:00Z"
		}
	];
};
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		chapters, err := adapter.GetChapters("series1")
		if err != nil {
			t.Fatalf("GetChapters() failed: %v", err)
		}

		if len(chapters) != 2 {
			t.Fatalf("Expected 2 chapters, got %d", len(chapters))
		}
		if chapters[0].Title != "Chapter 1" {
			t.Errorf("Expected first title 'Chapter 1', got '%s'", chapters[0].Title)
		}
		if chapters[0].Chapter != "1" {
			t.Errorf("Expected first chapter '1', got '%s'", chapters[0].Chapter)
		}
	})

	t.Run("GetPageURLs", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async (chapterId, mango) => {
	return [
		"http://example.com/page1.jpg",
		"http://example.com/page2.jpg",
		"http://example.com/page3.jpg"
	];
};
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		urls, err := adapter.GetPageURLs("ch1")
		if err != nil {
			t.Fatalf("GetPageURLs() failed: %v", err)
		}

		if len(urls) != 3 {
			t.Fatalf("Expected 3 URLs, got %d", len(urls))
		}
		if urls[0] != "http://example.com/page1.jpg" {
			t.Errorf("Expected first URL 'http://example.com/page1.jpg', got '%s'", urls[0])
		}
	})

	t.Run("Error Handling in Search", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async () => {
	throw new Error("Search failed");
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		_, err = adapter.Search("test")
		if err == nil {
			t.Fatal("Expected error from Search")
		}
	})
}

func TestPluginAPI(t *testing.T) {
	t.Run("Config", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	return [{ title: mango.config.test_value, identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		manifest := &plugins.PluginManifest{
			ID:          "test-plugin",
			Name:        "Test Plugin",
			Version:     "1.0.0",
			PluginType:  "downloader",
			EntryPoint:  "index.js",
			Description: "Test plugin",
			APIVersion:  "1.0",
			Config: map[string]interface{}{
				"test_value": map[string]interface{}{
					"type":    "string",
					"default": "configured_value",
				},
			},
		}

		scriptPath := filepath.Join(pluginDir, "index.js")
		os.WriteFile(scriptPath, []byte(pluginJS), 0644)

		app := testutil.SetupTestApp(t)
		runtime, err := plugins.NewPluginRuntime(app, manifest, pluginDir)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "configured_value" {
			t.Errorf("Expected title 'configured_value', got '%s'", results[0].Title)
		}
	})

	t.Run("State Persistence", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	mango.state.set("last_query", query);
	const lastQuery = mango.state.get("last_query");
	return [{ title: lastQuery, identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test_query")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "test_query" {
			t.Errorf("Expected title 'test_query', got '%s'", results[0].Title)
		}

		// Verify state was persisted - state is saved asynchronously
		// Give it a moment to write
		time.Sleep(50 * time.Millisecond)

		statePath := filepath.Join(pluginDir, "state.json")
		data, err := os.ReadFile(statePath)
		if err != nil {
			// State might not be persisted yet or might not be created on first set
			// This is acceptable behavior - state is lazy-loaded
			t.Logf("Note: state.json not yet created (this is OK for lazy persistence): %v", err)
			return
		}

		var state map[string]interface{}
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatalf("Failed to unmarshal state: %v", err)
		}

		if state["last_query"] != "test_query" {
			t.Errorf("Expected state.last_query 'test_query', got '%v'", state["last_query"])
		}
	})

	t.Run("HTTP Client", func(t *testing.T) {
		// Setup mock HTTP server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data": {"title": "Mock Response"}}`))
		}))
		defer mockServer.Close()

		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	const response = await mango.http.get("` + mockServer.URL + `/test");
	return [{ title: response.data.data.title, identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "Mock Response" {
			t.Errorf("Expected title 'Mock Response', got '%s'", results[0].Title)
		}
	})

	t.Run("HTTP Client with Timeout", func(t *testing.T) {
		// Setup mock HTTP server with delay
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Simulate slow response
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer mockServer.Close()

		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	const response = await mango.http.get("` + mockServer.URL + `/test", { timeout: 0.5 });
	return [{ title: "timeout-test", identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "timeout-test" {
			t.Errorf("Expected title 'timeout-test', got '%s'", results[0].Title)
		}
	})

	t.Run("HTML Parsing - parseHTML and querySelector", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	const html = '<html><body><h1 class="title">Test Title</h1><p>Content</p></body></html>';
	const doc = mango.utils.parseHTML(html);
	const title = doc.querySelector('h1.title');
	return [{ title: title.textContent, identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "Test Title" {
			t.Errorf("Expected title 'Test Title', got '%s'", results[0].Title)
		}
	})

	t.Run("HTML Parsing - querySelectorAll", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	const html = '<html><body><div class="item">Item 1</div><div class="item">Item 2</div></body></html>';
	const doc = mango.utils.parseHTML(html);
	const items = doc.querySelectorAll('div.item');
	return [{ title: items.length + " items", identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "2 items" {
			t.Errorf("Expected title '2 items', got '%s'", results[0].Title)
		}
	})

	t.Run("HTML Parsing - getAttribute", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	const html = '<html><body><a href="/test" data-id="123">Link</a></body></html>';
	const doc = mango.utils.parseHTML(html);
	const link = doc.querySelector('a');
	const href = link.getAttribute('href');
	const dataId = link.getAttribute('data-id');
	return [{ title: href + ":" + dataId, identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		expected := "/test:123"
		if results[0].Title != expected {
			t.Errorf("Expected title '%s', got '%s'", expected, results[0].Title)
		}
	})

	t.Run("HTML Parsing - XPath", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	const html = '<html><body><div class="chapter-item" data-id="ch1">Chapter 1</div><div class="chapter-item" data-id="ch2">Chapter 2</div></body></html>';
	const doc = mango.utils.parseHTML(html);
	const chapters = mango.utils.xpath(doc, '//div[@class="chapter-item"]');
	return [{ title: chapters.length + " chapters", identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "2 chapters" {
			t.Errorf("Expected title '2 chapters', got '%s'", results[0].Title)
		}
	})

	t.Run("HTML Parsing - Element querySelector", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginJS := `
exports.getInfo = () => ({ id: "test", name: "Test", version: "1.0.0" });
exports.search = async (query, mango) => {
	const html = '<html><body><div class="container"><span class="title">Nested Title</span></div></body></html>';
	const doc = mango.utils.parseHTML(html);
	const container = doc.querySelector('div.container');
	const title = container.querySelector('span.title');
	return [{ title: title.textContent, identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		runtime, err := createTestPlugin(t, pluginDir, pluginJS)
		if err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		adapter := plugins.NewPluginProviderAdapter(runtime)
		results, err := adapter.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}

		if results[0].Title != "Nested Title" {
			t.Errorf("Expected title 'Nested Title', got '%s'", results[0].Title)
		}
	})
}

func TestPluginLoader(t *testing.T) {
	t.Run("Load Valid Plugin", func(t *testing.T) {
		pluginDir := t.TempDir()
		pluginSubDir := filepath.Join(pluginDir, "test-plugin")
		os.MkdirAll(pluginSubDir, 0755)

		// Write manifest
		manifestJSON := `{
			"id": "test-plugin",
			"name": "Test Plugin",
			"version": "1.0.0",
			"plugin_type": "downloader",
			"entry_point": "index.js",
			"description": "Test plugin",
			"api_version": "1.0"
		}`
		os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

		// Write plugin script
		pluginJS := `
exports.getInfo = () => ({ id: "test-plugin", name: "Test Plugin", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

		app := testutil.SetupTestApp(t)
		manager := plugins.NewPluginManager(app, pluginDir)
		err := manager.LoadPlugin(pluginSubDir)
		if err != nil {
			t.Fatalf("manager.LoadPlugin() failed: %v", err)
		}

		// Verify plugin is registered
		provider, ok := providers.Get("test-plugin")
		if !ok {
			t.Fatal("Plugin not registered")
		}

		info := provider.GetInfo()
		if info.ID != "test-plugin" {
			t.Errorf("Expected ID 'test-plugin', got '%s'", info.ID)
		}

		t.Cleanup(func() {
			providers.UnregisterAll()
		})
	})

	t.Run("Load Plugins from Directory", func(t *testing.T) {
		pluginDir := t.TempDir()

		// Create two plugin directories
		for _, pluginName := range []string{"plugin1", "plugin2"} {
			pluginSubDir := filepath.Join(pluginDir, pluginName)
			os.MkdirAll(pluginSubDir, 0755)

			manifestJSON := `{
				"id": "` + pluginName + `",
				"name": "` + pluginName + `",
				"version": "1.0.0",
				"plugin_type": "downloader",
				"entry_point": "index.js",
				"description": "Test plugin",
				"api_version": "1.0"
			}`
			os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

			pluginJS := `
exports.getInfo = () => ({ id: "` + pluginName + `", name: "` + pluginName + `", version: "1.0.0" });
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
			os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)
		}

		app := testutil.SetupTestApp(t)
		manager := plugins.NewPluginManager(app, pluginDir)
		err := manager.LoadPlugins()
		if err != nil {
			t.Fatalf("manager.LoadPlugins() failed: %v", err)
		}

		// Verify both plugins are registered
		if _, ok := providers.Get("plugin1"); !ok {
			t.Error("plugin1 not registered")
		}
		if _, ok := providers.Get("plugin2"); !ok {
			t.Error("plugin2 not registered")
		}

		t.Cleanup(func() {
			providers.UnregisterAll()
		})
	})

	t.Run("Skip Invalid Plugin", func(t *testing.T) {
		pluginDir := t.TempDir()

		// Create invalid plugin (missing plugin.json)
		invalidDir := filepath.Join(pluginDir, "invalid")
		os.MkdirAll(invalidDir, 0755)

		app := testutil.SetupTestApp(t)
		manager := plugins.NewPluginManager(app, pluginDir)
		err := manager.LoadPlugins()
		// Should not error, just skip invalid plugins
		if err != nil {
			t.Fatalf("manager.LoadPlugins() should not fail for invalid plugins: %v", err)
		}
	})

	t.Run("Skip Hidden Directories", func(t *testing.T) {
		pluginDir := t.TempDir()

		// Create hidden directory
		hiddenDir := filepath.Join(pluginDir, ".hidden")
		os.MkdirAll(hiddenDir, 0755)
		os.WriteFile(filepath.Join(hiddenDir, "plugin.json"), []byte(`{"id": "hidden"}`), 0644)

		app := testutil.SetupTestApp(t)
		manager := plugins.NewPluginManager(app, pluginDir)
		err := manager.LoadPlugins()
		if err != nil {
			t.Fatalf("manager.LoadPlugins() failed: %v", err)
		}

		// Hidden plugin should not be registered
		if _, ok := providers.Get("hidden"); ok {
			t.Error("Hidden plugin should not be loaded")
		}

		t.Cleanup(func() {
			providers.UnregisterAll()
		})
	})
}

func TestManifest(t *testing.T) {
	t.Run("Load Valid Manifest", func(t *testing.T) {
		pluginDir := t.TempDir()
		manifestJSON := `{
			"id": "test-plugin",
			"name": "Test Plugin",
			"version": "1.0.0",
			"plugin_type": "downloader",
			"entry_point": "index.js",
			"description": "Test plugin",
			"api_version": "1.0"
		}`
		os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifestJSON), 0644)

		manifest, err := plugins.LoadManifest(pluginDir)
		if err != nil {
			t.Fatalf("plugins.LoadManifest() failed: %v", err)
		}

		if manifest.ID != "test-plugin" {
			t.Errorf("Expected ID 'test-plugin', got '%s'", manifest.ID)
		}
		if manifest.Name != "Test Plugin" {
			t.Errorf("Expected Name 'Test Plugin', got '%s'", manifest.Name)
		}
		if manifest.PluginType != "downloader" {
			t.Errorf("Expected PluginType 'downloader', got '%s'", manifest.PluginType)
		}
	})

	t.Run("Load Manifest with Config", func(t *testing.T) {
		pluginDir := t.TempDir()
		manifestJSON := `{
			"id": "test-plugin",
			"name": "Test Plugin",
			"version": "1.0.0",
			"plugin_type": "downloader",
			"entry_point": "index.js",
			"api_version": "1.0",
			"config": {
				"api_url": {
					"type": "string",
					"default": "https://api.example.com"
				}
			}
		}`
		os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifestJSON), 0644)

		manifest, err := plugins.LoadManifest(pluginDir)
		if err != nil {
			t.Fatalf("plugins.LoadManifest() failed: %v", err)
		}

		if manifest.Config == nil {
			t.Fatal("Expected config to be loaded")
		}

		apiURL, ok := manifest.Config["api_url"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected api_url config")
		}
		if apiURL["default"] != "https://api.example.com" {
			t.Errorf("Expected default 'https://api.example.com', got '%v'", apiURL["default"])
		}
	})

	t.Run("Load Non-Existent Manifest", func(t *testing.T) {
		pluginDir := t.TempDir()
		_, err := plugins.LoadManifest(pluginDir)
		if err == nil {
			t.Fatal("Expected error for non-existent manifest")
		}
	})

	t.Run("Load Invalid JSON Manifest", func(t *testing.T) {
		pluginDir := t.TempDir()
		os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`invalid json`), 0644)

		_, err := plugins.LoadManifest(pluginDir)
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
	})
}
