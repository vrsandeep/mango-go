package jobs

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

// JobContext is an interface that provides the necessary dependencies for a job to run.
// The core.App struct will implement this interface.
type JobContext interface {
	DB() *sql.DB
	Config() *config.Config
	WsHub() *websocket.Hub
	JobManager() *JobManager
}

// The task function signature now uses the interface.
type jobTask func(ctx JobContext)

type JobStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "idle", "running", "success", "failed"
	Message   string    `json:"message"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
}

type JobManager struct {
	mu      sync.Mutex
	jobs    map[string]jobTask
	status  map[string]*JobStatus
	running bool
	appCtx  JobContext // Store the app context for scheduled jobs
}

func NewManager(appCtx JobContext) *JobManager {
	jm := &JobManager{
		jobs:   make(map[string]jobTask),
		status: make(map[string]*JobStatus),
		appCtx: appCtx,
	}
	return jm
}

func (jm *JobManager) Register(name string, task jobTask) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.jobs[name] = task
	jm.status[name] = &JobStatus{Name: name, Status: "idle"}
}

// RunJob now accepts the JobContext interface.
func (jm *JobManager) RunJob(name string, ctx JobContext) error {
	jm.mu.Lock()
	if jm.running {
		jm.mu.Unlock()
		return fmt.Errorf("a job is already running")
	}

	task, ok := jm.jobs[name]
	if !ok {
		jm.mu.Unlock()
		return fmt.Errorf("job '%s' not found", name)
	}

	jm.running = true
	status := jm.status[name]
	status.Status = "running"
	status.StartTime = time.Now()
	status.Message = "Job started..."
	jm.mu.Unlock()

	log.Printf("Starting job: %s", name)
	// Run the actual task in a new goroutine so it doesn't block.
	go func() {
		defer func() {
			// Ensure we always update the status and unlock the manager
			if r := recover(); r != nil {
				log.Printf("Job '%s' panicked: %v", name, r)
				status.Status = "failed"
				status.Message = fmt.Sprintf("Job panicked: %v", r)
			}

			jm.mu.Lock()
			status.EndTime = time.Now()
			if status.Status == "running" { // If not already set to "failed"
				status.Status = "success"
				status.Message = "Job completed successfully."
			}
			jm.running = false
			jm.mu.Unlock()
			log.Printf("Finished job: %s", name)
		}()

		task(ctx)
	}()
	return nil
}


func (jm *JobManager) GetStatus() []*JobStatus {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	var statuses []*JobStatus
	for _, s := range jm.status {
		statuses = append(statuses, s)
	}
	return statuses
}