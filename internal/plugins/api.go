package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/xpath"
	"github.com/dop251/goja"
	"github.com/vrsandeep/mango-go/internal/core"
	"golang.org/x/net/html"
)

// MangoAPI provides the API that plugins can use.
type MangoAPI struct {
	app        *core.App
	pluginID   string
	statePath  string
	state      map[string]interface{}
	stateMu    sync.RWMutex
	stateDirty bool
	vm         *goja.Runtime // Current VM context
}

// NewMangoAPI creates a new Mango API instance for a plugin.
func NewMangoAPI(app *core.App, pluginID, pluginDir string) *MangoAPI {
	statePath := filepath.Join(pluginDir, "state.json")
	api := &MangoAPI{
		app:       app,
		pluginID:  pluginID,
		statePath: statePath,
		state:     make(map[string]interface{}),
	}

	// Load existing state
	api.loadState()

	// Start state persistence goroutine
	go api.persistStateLoop()

	return api
}

// Inject injects the Mango API into a goja runtime.
func (m *MangoAPI) Inject(vm *goja.Runtime) {
	m.vm = vm // Store VM for use in callbacks
	mango := vm.NewObject()

	// HTTP client
	httpObj := vm.NewObject()
	httpObj.Set("get", m.httpGet)
	httpObj.Set("post", m.httpPost)
	mango.Set("http", httpObj)

	// Logging
	logObj := vm.NewObject()
	logObj.Set("info", m.logInfo)
	logObj.Set("warn", m.logWarn)
	logObj.Set("error", m.logError)
	logObj.Set("debug", m.logDebug)
	mango.Set("log", logObj)

	// Configuration (will be set by loader)
	mango.Set("config", vm.NewObject())

	// State management
	stateObj := vm.NewObject()
	stateObj.Set("get", m.stateGet)
	stateObj.Set("set", m.stateSet)
	stateObj.Set("getAll", m.stateGetAll)
	stateObj.Set("clear", m.stateClear)
	mango.Set("state", stateObj)

	// Utilities
	utilsObj := vm.NewObject()
	utilsObj.Set("sanitizeFilename", m.sanitizeFilename)
	utilsObj.Set("parseHTML", m.parseHTML)
	utilsObj.Set("querySelector", m.querySelector)
	utilsObj.Set("querySelectorAll", m.querySelectorAll)
	utilsObj.Set("xpath", m.xpathQuery)
	mango.Set("utils", utilsObj)

	vm.Set("mango", mango)
}

// SetConfig sets the plugin configuration.
func (m *MangoAPI) SetConfig(vm *goja.Runtime, config map[string]interface{}) {
	mango := vm.Get("mango").ToObject(vm)
	configObj := vm.NewObject()
	for k, v := range config {
		configObj.Set(k, m.goToJS(vm, v))
	}
	mango.Set("config", configObj)
}

// httpGet performs an HTTP GET request.
func (m *MangoAPI) httpGet(call goja.FunctionCall) goja.Value {
	vm := m.vm
	url := call.Argument(0).String()

	if url == "" {
		vm.Interrupt("HTTP GET error: URL is required")
		return goja.Undefined()
	}

	var options map[string]interface{}
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) {
		options = call.Argument(1).ToObject(vm).Export().(map[string]interface{})
	}

	// Default timeout is 30 seconds
	timeout := 30 * time.Second
	if options != nil {
		if timeoutVal, ok := options["timeout"]; ok {
			if timeoutSec, ok := timeoutVal.(float64); ok {
				timeout = time.Duration(timeoutSec) * time.Second
			} else if timeoutSec, ok := timeoutVal.(int); ok {
				timeout = time.Duration(timeoutSec) * time.Second
			}
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP GET error: failed to create request for URL '%s': %v", url, err))
		return goja.Undefined()
	}

	// Set headers
	if options != nil {
		if headers, ok := options["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				req.Header.Set(k, fmt.Sprint(v))
			}
		}
	}

	// Add referer if not present
	if req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", url)
	}

	resp, err := client.Do(req)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP GET error: request to '%s' failed: %v", url, err))
		return goja.Undefined()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP GET error: failed to read response body from '%s': %v", url, err))
		return goja.Undefined()
	}

	// Try to parse as JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Not JSON, return as text
		data = string(body)
	}

	respObj := vm.NewObject()
	respObj.Set("status", resp.StatusCode)
	respObj.Set("statusText", resp.Status)
	respObj.Set("headers", m.headersToJS(vm, resp.Header))
	respObj.Set("data", m.goToJS(vm, data))
	respObj.Set("text", func() string { return string(body) })

	return respObj
}

// httpPost performs an HTTP POST request.
func (m *MangoAPI) httpPost(call goja.FunctionCall) goja.Value {
	vm := m.vm
	url := call.Argument(0).String()

	if url == "" {
		vm.Interrupt("HTTP POST error: URL is required")
		return goja.Undefined()
	}

	var bodyData interface{}
	var options map[string]interface{}

	if len(call.Arguments) > 1 {
		bodyData = call.Argument(1).Export()
	}
	if len(call.Arguments) > 2 && !goja.IsUndefined(call.Argument(2)) {
		options = call.Argument(2).ToObject(vm).Export().(map[string]interface{})
	}

	// Default timeout is 30 seconds
	timeout := 30 * time.Second
	if options != nil {
		if timeoutVal, ok := options["timeout"]; ok {
			if timeoutSec, ok := timeoutVal.(float64); ok {
				timeout = time.Duration(timeoutSec) * time.Second
			} else if timeoutSec, ok := timeoutVal.(int); ok {
				timeout = time.Duration(timeoutSec) * time.Second
			}
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	var body io.Reader
	contentType := "application/json"

	if bodyData != nil {
		if bodyStr, ok := bodyData.(string); ok {
			body = strings.NewReader(bodyStr)
			contentType = "text/plain"
		} else {
			jsonData, err := json.Marshal(bodyData)
			if err != nil {
				vm.Interrupt(fmt.Sprintf("HTTP POST error: failed to marshal request body for '%s': %v", url, err))
				return goja.Undefined()
			}
			body = strings.NewReader(string(jsonData))
		}
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP POST error: failed to create request for URL '%s': %v", url, err))
		return goja.Undefined()
	}

	req.Header.Set("Content-Type", contentType)

	if options != nil {
		if headers, ok := options["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				req.Header.Set(k, fmt.Sprint(v))
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP POST error: request to '%s' failed: %v", url, err))
		return goja.Undefined()
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP POST error: failed to read response body from '%s': %v", url, err))
		return goja.Undefined()
	}

	var data interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		data = string(respBody)
	}

	respObj := vm.NewObject()
	respObj.Set("status", resp.StatusCode)
	respObj.Set("statusText", resp.Status)
	respObj.Set("headers", m.headersToJS(vm, resp.Header))
	respObj.Set("data", m.goToJS(vm, data))
	respObj.Set("text", func() string { return string(respBody) })

	return respObj
}

// Logging functions
func (m *MangoAPI) logInfo(call goja.FunctionCall) goja.Value {
	_ = m.vm // store reference
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	log.Printf("[%s] INFO: %v", m.pluginID, fmt.Sprint(args...))
	return goja.Undefined()
}

func (m *MangoAPI) logWarn(call goja.FunctionCall) goja.Value {
	_ = m.vm
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	log.Printf("[%s] WARN: %v", m.pluginID, fmt.Sprint(args...))
	return goja.Undefined()
}

func (m *MangoAPI) logError(call goja.FunctionCall) goja.Value {
	_ = m.vm
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	log.Printf("[%s] ERROR: %v", m.pluginID, fmt.Sprint(args...))
	return goja.Undefined()
}

func (m *MangoAPI) logDebug(call goja.FunctionCall) goja.Value {
	_ = m.vm
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	log.Printf("[%s] DEBUG: %v", m.pluginID, fmt.Sprint(args...))
	return goja.Undefined()
}

// State management
func (m *MangoAPI) stateGet(call goja.FunctionCall) goja.Value {
	vm := m.vm
	key := call.Argument(0).String()

	m.stateMu.RLock()
	value, exists := m.state[key]
	m.stateMu.RUnlock()

	if !exists {
		return goja.Undefined()
	}

	return m.goToJS(vm, value)
}

func (m *MangoAPI) stateSet(call goja.FunctionCall) goja.Value {
	key := call.Argument(0).String()
	value := call.Argument(1).Export()

	m.stateMu.Lock()
	m.state[key] = value
	m.stateDirty = true
	m.stateMu.Unlock()

	return goja.Undefined()
}

func (m *MangoAPI) stateGetAll(call goja.FunctionCall) goja.Value {
	vm := m.vm

	m.stateMu.RLock()
	stateCopy := make(map[string]interface{})
	for k, v := range m.state {
		stateCopy[k] = v
	}
	m.stateMu.RUnlock()

	return m.goToJS(vm, stateCopy)
}

func (m *MangoAPI) stateClear(call goja.FunctionCall) goja.Value {
	m.stateMu.Lock()
	m.state = make(map[string]interface{})
	m.stateDirty = true
	m.stateMu.Unlock()

	return goja.Undefined()
}

// Utilities
func (m *MangoAPI) sanitizeFilename(call goja.FunctionCall) goja.Value {
	filename := call.Argument(0).String()
	// Basic sanitization - remove invalid characters
	sanitized := filename
	for _, char := range []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|'} {
		sanitized = strings.ReplaceAll(sanitized, string(char), "-")
	}
	return m.vm.ToValue(sanitized)
}

// GoToJS converts a Go value to a goja value (exported for use in runtime).
func (m *MangoAPI) GoToJS(vm *goja.Runtime, v interface{}) goja.Value {
	return m.goToJS(vm, v)
}

// Helper functions
func (m *MangoAPI) goToJS(vm *goja.Runtime, v interface{}) goja.Value {
	if v == nil {
		return goja.Null()
	}

	switch val := v.(type) {
	case string:
		return vm.ToValue(val)
	case int:
		return vm.ToValue(val)
	case int64:
		return vm.ToValue(int(val))
	case float64:
		return vm.ToValue(val)
	case bool:
		return vm.ToValue(val)
	case []interface{}:
		arr := vm.NewArray(len(val))
		for i, item := range val {
			arr.Set(fmt.Sprintf("%d", i), m.goToJS(vm, item))
		}
		return arr
	case map[string]interface{}:
		obj := vm.NewObject()
		for k, v := range val {
			obj.Set(k, m.goToJS(vm, v))
		}
		return obj
	default:
		return vm.ToValue(val)
	}
}

func (m *MangoAPI) headersToJS(vm *goja.Runtime, headers http.Header) goja.Value {
	headerMap := make(map[string]interface{})
	for k, v := range headers {
		if len(v) > 0 {
			headerMap[k] = v[0]
		}
	}
	return m.goToJS(vm, headerMap)
}

// HTML parsing utilities

// DocumentWrapper wraps a goquery document for JavaScript access
type DocumentWrapper struct {
	doc     *goquery.Document
	htmlStr string // Store original HTML for XPath queries
	vm      *goja.Runtime
	api     *MangoAPI
}

// parseHTML parses HTML string and returns a document object
func (m *MangoAPI) parseHTML(call goja.FunctionCall) goja.Value {
	vm := m.vm
	htmlStr := call.Argument(0).String()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		vm.Interrupt(fmt.Sprintf("parseHTML error: failed to parse HTML: %v", err))
		return goja.Undefined()
	}

	wrapper := &DocumentWrapper{
		doc:     doc,
		htmlStr: htmlStr,
		vm:      vm,
		api:     m,
	}

	docObj := vm.NewObject()
	docObj.Set("querySelector", func(selector string) goja.Value {
		return wrapper.querySelector(selector)
	})
	docObj.Set("querySelectorAll", func(selector string) goja.Value {
		return wrapper.querySelectorAll(selector)
	})
	// Store HTML string for XPath queries
	docObj.Set("_html", htmlStr)

	return docObj
}

func (d *DocumentWrapper) querySelector(selector string) goja.Value {
	selection := d.doc.Find(selector).First()
	if selection.Length() == 0 {
		return d.vm.ToValue(nil)
	}
	return d.api.elementToJS(d.vm, selection)
}

func (d *DocumentWrapper) querySelectorAll(selector string) goja.Value {
	selection := d.doc.Find(selector)
	return d.api.selectionToJS(d.vm, selection)
}

// querySelector finds a single element using CSS selector
func (m *MangoAPI) querySelector(call goja.FunctionCall) goja.Value {
	vm := m.vm
	docVal := call.Argument(0)
	selector := call.Argument(1).String()

	// Try to get the document from the wrapper
	docObj := docVal.ToObject(vm)
	if docObj == nil {
		vm.Interrupt("querySelector error: invalid document object")
		return goja.Undefined()
	}

	querySelectorFn := docObj.Get("querySelector")
	if querySelectorFn == nil || goja.IsUndefined(querySelectorFn) {
		vm.Interrupt("querySelector error: document object does not have querySelector method")
		return goja.Undefined()
	}

	if fn, ok := goja.AssertFunction(querySelectorFn); ok {
		result, err := fn(docObj, vm.ToValue(selector))
		if err != nil {
			vm.Interrupt(fmt.Sprintf("querySelector error: %v", err))
			return goja.Undefined()
		}
		return result
	}

	vm.Interrupt("querySelector error: querySelector is not a function")
	return goja.Undefined()
}

// querySelectorAll finds all elements matching CSS selector
func (m *MangoAPI) querySelectorAll(call goja.FunctionCall) goja.Value {
	vm := m.vm
	docVal := call.Argument(0)
	selector := call.Argument(1).String()

	// Try to get the document from the wrapper
	docObj := docVal.ToObject(vm)
	if docObj == nil {
		vm.Interrupt("querySelectorAll error: invalid document object")
		return goja.Undefined()
	}

	querySelectorAllFn := docObj.Get("querySelectorAll")
	if querySelectorAllFn == nil || goja.IsUndefined(querySelectorAllFn) {
		vm.Interrupt("querySelectorAll error: document object does not have querySelectorAll method")
		return goja.Undefined()
	}

	if fn, ok := goja.AssertFunction(querySelectorAllFn); ok {
		result, err := fn(docObj, vm.ToValue(selector))
		if err != nil {
			vm.Interrupt(fmt.Sprintf("querySelectorAll error: %v", err))
			return goja.Undefined()
		}
		return result
	}

	vm.Interrupt("querySelectorAll error: querySelectorAll is not a function")
	return goja.Undefined()
}

// xpathQuery executes an XPath query on a document
func (m *MangoAPI) xpathQuery(call goja.FunctionCall) goja.Value {
	vm := m.vm

	// Get HTML string from document or first argument
	htmlStr := ""
	xpathExpr := ""

	if len(call.Arguments) >= 2 {
		// First arg is document, second is XPath expression
		docVal := call.Argument(0)
		xpathExpr = call.Argument(1).String()

		// Try to get HTML string from document object
		docObj := docVal.ToObject(vm)
		if docObj != nil {
			if htmlVal := docObj.Get("_html"); !goja.IsUndefined(htmlVal) {
				htmlStr = htmlVal.String()
			}
		}
	} else if len(call.Arguments) == 2 {
		// Alternative: HTML string as first arg, XPath as second
		htmlStr = call.Argument(0).String()
		xpathExpr = call.Argument(1).String()
	} else {
		vm.Interrupt("xpath error: requires document (from parseHTML) and XPath expression, or HTML string and XPath expression")
		return goja.Undefined()
	}

	if htmlStr == "" || xpathExpr == "" {
		vm.Interrupt("xpath error: HTML string and XPath expression are required")
		return goja.Undefined()
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		vm.Interrupt(fmt.Sprintf("xpath error: failed to parse HTML: %v", err))
		return goja.Undefined()
	}

	// Create XPath navigator for HTML
	nav := createHTMLNavigator(doc)

	// Compile XPath expression
	expr, err := xpath.Compile(xpathExpr)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("xpath error: failed to compile XPath expression '%s': %v", xpathExpr, err))
		return goja.Undefined()
	}

	// Execute query
	iter := expr.Select(nav)
	var nodes []*html.Node
	for iter.MoveNext() {
		if htmlNav, ok := iter.Current().(*htmlNavigator); ok {
			nodes = append(nodes, htmlNav.node)
		}
	}

	// Convert nodes to JavaScript elements using goquery
	var elements []goja.Value
	for _, node := range nodes {
		selection := goquery.NewDocumentFromNode(node).Selection
		elements = append(elements, m.elementToJS(vm, selection))
	}

	arr := vm.NewArray(len(elements))
	for i, elem := range elements {
		arr.Set(fmt.Sprintf("%d", i), elem)
	}
	return arr
}

// elementToJS converts a goquery selection to a JavaScript element object
func (m *MangoAPI) elementToJS(vm *goja.Runtime, selection *goquery.Selection) goja.Value {
	element := vm.NewObject()

	// textContent property
	element.Set("textContent", selection.Text())

	// innerHTML property
	html, _ := selection.Html()
	element.Set("innerHTML", html)

	// getAttribute method
	element.Set("getAttribute", func(name string) goja.Value {
		val, exists := selection.Attr(name)
		if !exists {
			return goja.Undefined()
		}
		return vm.ToValue(val)
	})

	// querySelector method (for finding child elements)
	element.Set("querySelector", func(selector string) goja.Value {
		child := selection.Find(selector).First()
		if child.Length() == 0 {
			return vm.ToValue(nil)
		}
		return m.elementToJS(vm, child)
	})

	// querySelectorAll method (for finding all child elements)
	element.Set("querySelectorAll", func(selector string) goja.Value {
		children := selection.Find(selector)
		return m.selectionToJS(vm, children)
	})

	return element
}

// selectionToJS converts a goquery selection (multiple elements) to a JavaScript array
func (m *MangoAPI) selectionToJS(vm *goja.Runtime, selection *goquery.Selection) goja.Value {
	var elements []goja.Value
	selection.Each(func(i int, s *goquery.Selection) {
		elements = append(elements, m.elementToJS(vm, s))
	})

	arr := vm.NewArray(len(elements))
	for i, elem := range elements {
		arr.Set(fmt.Sprintf("%d", i), elem)
	}
	return arr
}

// htmlNavigator implements xpath.NodeNavigator for HTML nodes
type htmlNavigator struct {
	node *html.Node
	pos  int
}

func createHTMLNavigator(root *html.Node) *htmlNavigator {
	return &htmlNavigator{node: root, pos: 0}
}

func (h *htmlNavigator) NodeType() xpath.NodeType {
	switch h.node.Type {
	case html.DocumentNode:
		return xpath.RootNode
	case html.ElementNode:
		// Check if we're on an attribute (pos > 0 means we're iterating attributes)
		if h.pos > 0 && h.pos <= len(h.node.Attr) {
			return xpath.AttributeNode
		}
		return xpath.ElementNode
	case html.TextNode:
		return xpath.TextNode
	case html.CommentNode:
		return xpath.CommentNode
	default:
		return xpath.ElementNode
	}
}

func (h *htmlNavigator) LocalName() string {
	if h.node.Type == html.ElementNode {
		// If we're on an attribute
		if h.pos > 0 && h.pos <= len(h.node.Attr) {
			return h.node.Attr[h.pos-1].Key
		}
		return h.node.Data
	}
	return ""
}

func (h *htmlNavigator) Prefix() string {
	return ""
}

func (h *htmlNavigator) Value() string {
	switch h.node.Type {
	case html.TextNode:
		return h.node.Data
	case html.CommentNode:
		return h.node.Data
	case html.ElementNode:
		// If we're on an attribute
		if h.pos > 0 && h.pos <= len(h.node.Attr) {
			return h.node.Attr[h.pos-1].Val
		}
	}
	return ""
}

func (h *htmlNavigator) Copy() xpath.NodeNavigator {
	return &htmlNavigator{node: h.node, pos: h.pos}
}

func (h *htmlNavigator) MoveToRoot() {
	for h.node.Parent != nil {
		h.node = h.node.Parent
	}
	h.pos = 0
}

func (h *htmlNavigator) MoveToParent() bool {
	if h.node.Parent != nil {
		h.node = h.node.Parent
		h.pos = 0
		return true
	}
	return false
}

func (h *htmlNavigator) MoveToNextAttribute() bool {
	if h.node.Type == html.ElementNode && h.pos < len(h.node.Attr) {
		h.pos++
		return true
	}
	return false
}

func (h *htmlNavigator) MoveToChild() bool {
	if h.node.FirstChild != nil {
		h.node = h.node.FirstChild
		h.pos = 0
		return true
	}
	return false
}

func (h *htmlNavigator) MoveToFirst() bool {
	if h.node.Parent != nil && h.node.Parent.FirstChild != nil {
		h.node = h.node.Parent.FirstChild
		h.pos = 0
		return true
	}
	return false
}

func (h *htmlNavigator) String() string {
	return h.Value()
}

func (h *htmlNavigator) MoveToNext() bool {
	if h.node.NextSibling != nil {
		h.node = h.node.NextSibling
		h.pos = 0
		return true
	}
	return false
}

func (h *htmlNavigator) MoveToPrevious() bool {
	if h.node.PrevSibling != nil {
		h.node = h.node.PrevSibling
		h.pos = 0
		return true
	}
	return false
}

func (h *htmlNavigator) MoveTo(other xpath.NodeNavigator) bool {
	if o, ok := other.(*htmlNavigator); ok {
		h.node = o.node
		h.pos = o.pos
		return true
	}
	return false
}

// State persistence
func (m *MangoAPI) loadState() {
	if data, err := os.ReadFile(m.statePath); err == nil {
		json.Unmarshal(data, &m.state)
	}
}

func (m *MangoAPI) persistStateLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		m.stateMu.Lock()
		dirty := m.stateDirty
		if dirty {
			m.stateDirty = false
			stateCopy := make(map[string]interface{})
			for k, v := range m.state {
				stateCopy[k] = v
			}
			m.stateMu.Unlock()

			if data, err := json.Marshal(stateCopy); err == nil {
				os.WriteFile(m.statePath, data, 0600)
			}
		} else {
			m.stateMu.Unlock()
		}
	}
}
