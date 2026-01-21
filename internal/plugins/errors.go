package plugins

import "fmt"

// PluginError represents an error that occurred in a plugin.
type PluginError struct {
	PluginID  string
	Function  string
	Message   string
	Cause     error
	IsTimeout bool
	IsPanic   bool
}

func (e *PluginError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("plugin %s: function %s: %s: %v", e.PluginID, e.Function, e.Message, e.Cause)
	}
	return fmt.Sprintf("plugin %s: function %s: %s", e.PluginID, e.Function, e.Message)
}

func (e *PluginError) Unwrap() error {
	return e.Cause
}
