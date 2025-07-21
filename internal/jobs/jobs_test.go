package jobs_test

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/stretchr/testify/assert"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

// mockJobContext implements JobContext for testing
// Only implements the methods needed for these tests
type mockJobContext struct {
	db        *sql.DB
	config    *config.Config
	wsHub     *websocket.Hub
	jobMgr    *jobs.JobManager
}

func (m *mockJobContext) DB() *sql.DB              { return m.db }
func (m *mockJobContext) Config() *config.Config   { return m.config }
func (m *mockJobContext) WsHub() *websocket.Hub   { return m.wsHub }
func (m *mockJobContext) JobManager() *jobs.JobManager  { return m.jobMgr }

func TestNewManagerAndRegister(t *testing.T) {
	ctx := &mockJobContext{config: &config.Config{}, wsHub: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	assert.NotNil(t, mgr)
	mgr.Register("test", "Test", func(ctx jobs.JobContext) {})
	statuses := mgr.GetStatus()
	var found bool
	for _, s := range statuses {
		if s.ID == "test" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find job named 'test' in statuses")
}

func TestRunJob_Success(t *testing.T) {
	ctx := &mockJobContext{wsHub: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	ctx.jobMgr = mgr
	var called bool
	mgr.Register("job1", "Job 1", func(ctx jobs.JobContext) { called = true })
	err := mgr.RunJob("job1", ctx)
	assert.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, called)
	statuses := mgr.GetStatus()
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, "job1", statuses[0].ID)
}

func TestRunJob_AlreadyRunning(t *testing.T) {
	ctx := &mockJobContext{wsHub: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	ctx.jobMgr = mgr
	block := make(chan struct{})
	mgr.Register("job1", "Job 1", func(ctx jobs.JobContext) { <-block })
	_ = mgr.RunJob("job1", ctx)
	err := mgr.RunJob("job1", ctx)
	assert.Error(t, err)
	close(block)
}

func TestRunJob_NotFound(t *testing.T) {
	ctx := &mockJobContext{wsHub: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	err := mgr.RunJob("nope", ctx)
	assert.Error(t, err)
}

func TestGetStatus(t *testing.T) {
	ctx := &mockJobContext{wsHub: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	mgr.Register("job1", "Job 1", func(ctx jobs.JobContext) {})
	mgr.Register("job2", "Job 2", func(ctx jobs.JobContext) {})
	statuses := mgr.GetStatus()
	assert.Len(t, statuses, 2)
}

func TestStartJobs_SchedulesJob(t *testing.T) {
	// Use a short interval for test
	mgr := jobs.NewManager(nil)
	var mu sync.Mutex
	var triggered int
	mgr.Register("library-sync", "Sync", func(ctx jobs.JobContext) {
		mu.Lock()
		triggered++
		mu.Unlock()
	})
	ctx := &mockJobContext{config: &config.Config{ScanInterval: 1}, jobMgr: mgr, wsHub: websocket.NewHub()}
	jobs.StartJobs(ctx)
	time.Sleep(1200 * time.Millisecond)
	mu.Lock()
	count := triggered
	mu.Unlock()
	assert.GreaterOrEqual(t, count, 1)
}

func TestStartLibrarySyncJob_Disabled(t *testing.T) {
	mgr := jobs.NewManager(nil)
	ctx := &mockJobContext{config: &config.Config{ScanInterval: 0}, jobMgr: mgr, wsHub: websocket.NewHub()}
	s := gocron.NewScheduler(time.UTC)
	jobs.StartJobs(ctx)
	// No panic, no job scheduled
	assert.Equal(t, 0, len(s.Jobs()))
}
