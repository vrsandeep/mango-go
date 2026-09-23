package plugins

import (
	"errors"
	"sync"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// ErrAutoUpdateInProgress is returned when an automatic update run is requested
// while another one is still running.
var ErrAutoUpdateInProgress = errors.New("an automatic plugin update is already in progress")

// autoUpdateRunMu allows only one automatic update run at a time. Callers that
// find it held return immediately instead of queueing a redundant run.
var autoUpdateRunMu sync.Mutex

// pluginInstallLocks guards the filesystem, plugin manager and database writes
// performed while installing a single plugin, keyed by plugin ID. Installs of
// different plugins touch different directories and rows, so they may proceed
// concurrently. This is separate from autoUpdateRunMu because an automatic run
// installs plugins while holding that lock, and Go mutexes are not reentrant.
var pluginInstallLocks sync.Map

// lockPluginInstall blocks until the given plugin can be installed, and returns
// the function that releases it.
func lockPluginInstall(pluginID string) func() {
	value, _ := pluginInstallLocks.LoadOrStore(pluginID, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

var (
	autoUpdateStatusMu sync.RWMutex
	autoUpdateStatus   models.PluginAutoUpdateStatus
)

// AutoUpdateStatus returns the outcome of the most recent automatic update run.
func AutoUpdateStatus() models.PluginAutoUpdateStatus {
	autoUpdateStatusMu.RLock()
	defer autoUpdateStatusMu.RUnlock()

	status := autoUpdateStatus
	status.Updated = append(make([]models.PluginUpdateInfo, 0, len(autoUpdateStatus.Updated)), autoUpdateStatus.Updated...)
	status.Failed = append(make([]models.PluginUpdateFailure, 0, len(autoUpdateStatus.Failed)), autoUpdateStatus.Failed...)
	return status
}

// markAutoUpdateRunning flags a run as in progress while keeping the previous
// outcome visible until the new one is recorded.
func markAutoUpdateRunning() {
	autoUpdateStatusMu.Lock()
	defer autoUpdateStatusMu.Unlock()

	autoUpdateStatus.Running = true
}

// recordAutoUpdateResult stores the outcome of a finished run.
func recordAutoUpdateResult(result *models.PluginAutoUpdateResult, err error) {
	autoUpdateStatusMu.Lock()
	defer autoUpdateStatusMu.Unlock()

	autoUpdateStatus = models.PluginAutoUpdateStatus{
		Running:   false,
		HasRun:    true,
		LastRunAt: time.Now(),
	}

	if err != nil {
		autoUpdateStatus.Error = err.Error()
		return
	}

	autoUpdateStatus.Updated = result.Updated
	autoUpdateStatus.Failed = result.Failed
}
