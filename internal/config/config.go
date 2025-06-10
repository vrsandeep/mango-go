// This file defines the configuration structure for the application.
// For now, it's very simple, only holding the path to the manga library.
package config

// Config holds all configuration settings for the application.
type Config struct {
	LibraryPath string `json:"library_path"`
	// We can add more settings here later, like port number, database path, etc.
}
