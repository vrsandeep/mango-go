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

	"github.com/dop251/goja"
	"github.com/vrsandeep/mango-go/internal/core"
)

// MangoAPI provides the API that plugins can use.
type MangoAPI struct {
	app       *core.App
	pluginID  string
	statePath string
	state     map[string]interface{}
	stateMu   sync.RWMutex
	stateDirty bool
	vm        *goja.Runtime // Current VM context
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

	// Utilities (basic for now)
	utilsObj := vm.NewObject()
	utilsObj.Set("sanitizeFilename", m.sanitizeFilename)
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

	var options map[string]interface{}
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) {
		options = call.Argument(1).ToObject(vm).Export().(map[string]interface{})
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP request error: %v", err))
		return goja.Undefined()
	}

	// Set headers
	if headers, ok := options["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}

	// Add referer if not present
	if req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", url)
	}

	resp, err := client.Do(req)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP request failed: %v", err))
		return goja.Undefined()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("Failed to read response body: %v", err))
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

	var bodyData interface{}
	var options map[string]interface{}

	if len(call.Arguments) > 1 {
		bodyData = call.Argument(1).Export()
	}
	if len(call.Arguments) > 2 && !goja.IsUndefined(call.Argument(2)) {
		options = call.Argument(2).ToObject(vm).Export().(map[string]interface{})
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var body io.Reader
	contentType := "application/json"

	if bodyData != nil {
		if bodyStr, ok := bodyData.(string); ok {
			body = strings.NewReader(bodyStr)
			contentType = "text/plain"
		} else {
			jsonData, _ := json.Marshal(bodyData)
			body = strings.NewReader(string(jsonData))
		}
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP request error: %v", err))
		return goja.Undefined()
	}

	req.Header.Set("Content-Type", contentType)

	if headers, ok := options["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("HTTP request failed: %v", err))
		return goja.Undefined()
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		vm.Interrupt(fmt.Sprintf("Failed to read response body: %v", err))
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

