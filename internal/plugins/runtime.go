package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dop251/goja"
	"github.com/vrsandeep/mango-go/internal/core"
)

// PluginRuntime manages a goja VM for a plugin.
type PluginRuntime struct {
	vm       *goja.Runtime
	manifest *PluginManifest
	api      *MangoAPI
	pluginDir string
}

// NewPluginRuntime creates a new plugin runtime.
func NewPluginRuntime(app *core.App, manifest *PluginManifest, pluginDir string) (*PluginRuntime, error) {
	vm := goja.New()

	// Create Mango API
	api := NewMangoAPI(app, manifest.ID, pluginDir)
	api.Inject(vm)

	// Load plugin config
	config := make(map[string]interface{})
	if manifest.Config != nil {
		for k, v := range manifest.Config {
			if configObj, ok := v.(map[string]interface{}); ok {
				if defaultVal, ok := configObj["default"]; ok {
					config[k] = defaultVal
				}
			} else {
				config[k] = v
			}
		}
	}
	api.SetConfig(vm, config)

	// Load plugin script
	scriptPath := filepath.Join(pluginDir, manifest.EntryPoint)
	scriptData, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin script: %w", err)
	}

	// Create CommonJS-like exports object
	exports := vm.NewObject()
	vm.Set("exports", exports)

	// Wrap script in a function to provide module-like context
	// This simulates CommonJS module wrapper: (function(exports, require, module) { ... })
	moduleScript := fmt.Sprintf(`
		(function(exports) {
			%s
		})(exports);
	`, string(scriptData))

	// Execute plugin script
	_, err = vm.RunString(moduleScript)
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin script: %w", err)
	}

	// Verify required exports
	exportsVal := vm.Get("exports")
	if exportsVal == nil || goja.IsUndefined(exportsVal) || goja.IsNull(exportsVal) {
		return nil, fmt.Errorf("plugin does not export 'exports' object")
	}

	exportsObj := exportsVal.ToObject(vm)
	requiredExports := []string{"getInfo", "search", "getChapters", "getPageURLs"}
	for _, exp := range requiredExports {
		if exportsObj.Get(exp) == nil {
			return nil, fmt.Errorf("plugin missing required export: %s", exp)
		}
	}

	return &PluginRuntime{
		vm:        vm,
		manifest:  manifest,
		api:       api,
		pluginDir: pluginDir,
	}, nil
}

// Manifest returns the plugin manifest (for testing)
func (r *PluginRuntime) Manifest() *PluginManifest {
	return r.manifest
}

// VM returns the goja runtime (for testing)
func (r *PluginRuntime) VM() *goja.Runtime {
	return r.vm
}

// Call calls a plugin function with error recovery.
func (r *PluginRuntime) Call(functionName string, args ...interface{}) (goja.Value, error) {
	return r.CallWithContext(context.Background(), functionName, args...)
}

// CallWithContext calls a plugin function with a context for timeout.
func (r *PluginRuntime) CallWithContext(ctx context.Context, functionName string, args ...interface{}) (goja.Value, error) {
	exports := r.vm.Get("exports")
	if exports == nil {
		return nil, fmt.Errorf("exports not found")
	}

	exportsObj := exports.ToObject(r.vm)
	fn := exportsObj.Get(functionName)
	if fn == nil {
		return nil, fmt.Errorf("function %s not found", functionName)
	}

	if !goja.IsUndefined(fn) && !goja.IsNull(fn) {
		if callable, ok := goja.AssertFunction(fn); ok {
			// Convert args to goja values
			gojaArgs := make([]goja.Value, len(args))
			for i, arg := range args {
				gojaArgs[i] = r.api.GoToJS(r.vm, arg)
			}

			// Add mango API as last argument
			mango := r.vm.Get("mango")
			gojaArgs = append(gojaArgs, mango)

			// Call with timeout
			done := make(chan goja.Value, 1)
			errChan := make(chan error, 1)

			// Capture manifest for use in goroutine
			manifestID := r.manifest.ID

			go func() {
				defer func() {
					if panicVal := recover(); panicVal != nil {
						errChan <- &PluginError{
							PluginID: manifestID,
							Function: functionName,
							Message:  fmt.Sprintf("panic: %v", panicVal),
							IsPanic:  true,
						}
					}
				}()

				val, err := callable(goja.Undefined(), gojaArgs...)
				if err != nil {
					errChan <- &PluginError{
						PluginID: manifestID,
						Function: functionName,
						Message:  err.Error(),
						Cause:    err,
					}
					return
				}

				// Check if the result is a Promise (goja represents promises as objects)
				if !goja.IsUndefined(val) && !goja.IsNull(val) {
					if promiseObj := val.ToObject(r.vm); promiseObj != nil {
						// Check if it has promise-like properties (then method)
						then := promiseObj.Get("then")
						if then != nil && !goja.IsUndefined(then) {
							// It's a promise, we need to await it
							resultChan := make(chan goja.Value, 1)
							errorChan := make(chan error, 1)
							promiseResolved := false

							// Create resolve and reject handlers as JavaScript functions
							resolveHandler := r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
								if !promiseResolved {
									promiseResolved = true
									resultChan <- call.Argument(0)
								}
								return goja.Undefined()
							})

							rejectHandler := r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
								if !promiseResolved {
									promiseResolved = true
									errVal := call.Argument(0)
									var errMsg string
									if !goja.IsUndefined(errVal) && !goja.IsNull(errVal) {
										errMsg = errVal.String()
									}
									if errMsg == "" {
										errMsg = "unknown error"
									}
									errorChan <- fmt.Errorf("promise rejected: %s", errMsg)
								}
								return goja.Undefined()
							})

							// Call then(resolve, reject) on the promise
							if thenFn, ok := goja.AssertFunction(then); ok {
								_, callErr := thenFn(promiseObj, resolveHandler, rejectHandler)
								if callErr != nil {
									errorChan <- fmt.Errorf("failed to handle promise: %w", callErr)
									promiseResolved = true
								}

								// Wait for promise resolution
								select {
								case result := <-resultChan:
									done <- result
									return
								case err := <-errorChan:
									errChan <- &PluginError{
										PluginID: manifestID,
										Function: functionName,
										Message:  err.Error(),
										Cause:    err,
									}
									return
								case <-time.After(30 * time.Second):
									errChan <- &PluginError{
										PluginID:  manifestID,
										Function:  functionName,
										Message:   "promise timeout",
										IsTimeout: true,
									}
									return
								}
							}
						}
					}
				}

				done <- val
			}()

			select {
			case val := <-done:
				return val, nil
			case err := <-errChan:
				return nil, err
			case <-ctx.Done():
				return nil, &PluginError{
					PluginID:  manifestID,
					Function:  functionName,
					Message:   "timeout",
					IsTimeout: true,
				}
			case <-time.After(30 * time.Second):
				return nil, &PluginError{
					PluginID:  manifestID,
					Function:  functionName,
					Message:   "timeout after 30 seconds",
					IsTimeout: true,
				}
			}
		}
	}

	return nil, fmt.Errorf("function %s is not callable", functionName)
}

