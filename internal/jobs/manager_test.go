package jobs_test

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

type fakeJobContext struct {
	db     *sql.DB
	cfg    *config.Config
	ws     *websocket.Hub
	jobMgr *jobs.JobManager
}

func (f *fakeJobContext) DB() *sql.DB                  { return f.db }
func (f *fakeJobContext) Config() *config.Config       { return f.cfg }
func (f *fakeJobContext) WsHub() *websocket.Hub        { return f.ws }
func (f *fakeJobContext) JobManager() *jobs.JobManager { return f.jobMgr }

func TestManager_NewManager(t *testing.T) {
	ctx := &fakeJobContext{cfg: &config.Config{}, ws: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	assert.NotNil(t, mgr)
	assert.Empty(t, mgr.GetStatus())
}

func TestManager_RegisterAndGetStatus(t *testing.T) {
	ctx := &fakeJobContext{cfg: &config.Config{}, ws: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	mgr.Register("jobA", "Job A", func(ctx jobs.JobContext) {})
	mgr.Register("jobB", "Job B", func(ctx jobs.JobContext) {})
	statuses := mgr.GetStatus()
	assert.Len(t, statuses, 2)
	var foundA, foundB bool
	for _, s := range statuses {
		if s.ID == "jobA" {
			foundA = true
		}
		if s.ID == "jobB" {
			foundB = true
		}
	}
	assert.True(t, foundA && foundB)
}

func TestManager_RunJob_SuccessAndStatus(t *testing.T) {
	ctx := &fakeJobContext{cfg: &config.Config{}, ws: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	ctx.jobMgr = mgr
	var called bool
	mgr.Register("jobX", "Job X", func(ctx jobs.JobContext) { called = true })
	err := mgr.RunJob("jobX", ctx)
	assert.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, called)
	statuses := mgr.GetStatus()
	assert.Equal(t, "success", statuses[0].Status)
}

func TestManager_RunJob_AlreadyRunning(t *testing.T) {
	ctx := &fakeJobContext{cfg: &config.Config{}, ws: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	ctx.jobMgr = mgr
	block := make(chan struct{})
	mgr.Register("jobY", "Job Y", func(ctx jobs.JobContext) { <-block })
	_ = mgr.RunJob("jobY", ctx)
	err := mgr.RunJob("jobY", ctx)
	assert.Error(t, err)
	close(block)
}

func TestManager_RunJob_NotFound(t *testing.T) {
	ctx := &fakeJobContext{cfg: &config.Config{}, ws: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	err := mgr.RunJob("nojob", ctx)
	assert.Error(t, err)
}

func TestManager_RunJob_Panic(t *testing.T) {
	ctx := &fakeJobContext{cfg: &config.Config{}, ws: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	ctx.jobMgr = mgr
	mgr.Register("panicJob", "Panic Job", func(ctx jobs.JobContext) { panic("fail") })
	err := mgr.RunJob("panicJob", ctx)
	assert.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	statuses := mgr.GetStatus()
	assert.Equal(t, "failed", statuses[0].Status)
	assert.Contains(t, statuses[0].Message, "panicked")
}

func TestManager_Concurrency(t *testing.T) {
	ctx := &fakeJobContext{cfg: &config.Config{}, ws: websocket.NewHub()}
	mgr := jobs.NewManager(ctx)
	ctx.jobMgr = mgr
	var mu sync.Mutex
	var count int
	mgr.Register("jobC", "Job C", func(ctx jobs.JobContext) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			_ = mgr.RunJob("jobC", ctx)
			wg.Done()
		}()
	}
	wg.Wait()
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	assert.Equal(t, 1, count, "job should only run once concurrently")
	if count != 1 {
		statuses := mgr.GetStatus()
		for _, s := range statuses {
			t.Logf("Job %s: %s", s.ID, s.Status)
		}
	}
	mu.Unlock()
}
