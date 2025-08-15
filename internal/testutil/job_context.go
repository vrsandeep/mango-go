// This file contains shared test utilities for job context mocking.

package testutil

import (
	"database/sql"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

// MockJobContext implements jobs.JobContext for testing
type MockJobContext struct {
	App *core.App
}

func (m *MockJobContext) DB() *sql.DB                  { return m.App.DB() }
func (m *MockJobContext) Config() *config.Config       { return m.App.Config() }
func (m *MockJobContext) WsHub() *websocket.Hub        { return websocket.NewHub() }
func (m *MockJobContext) JobManager() *jobs.JobManager { return nil }
