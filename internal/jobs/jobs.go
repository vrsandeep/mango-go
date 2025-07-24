package jobs

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

// StartJobs starts the background job scheduler.
func StartJobs(app JobContext) {
	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()

	startLibrarySyncJob(s, app)

	log.Println("Starting background job scheduler...")
	s.StartAsync()
}

func startLibrarySyncJob(s *gocron.Scheduler, app JobContext) {
	interval := app.Config().ScanInterval
	if interval == 0 {
		log.Println("Library sync interval is 0, scheduled sync is disabled.")
		return
	}

	jobId := "library-sync"
	log.Printf("Scheduling job: '%s' to run every %d minutes.", jobId, interval)

	_, err := s.Every(interval).Minutes().Do(func() {
		log.Println("Scheduler is triggering job:", jobId)
		// Submit the job to the manager instead of running it directly.
		// This prevents conflicts with manually triggered jobs.
		err := app.JobManager().RunJob(jobId, app)
		if err != nil {
			log.Printf("Scheduled job '%s' could not start: %v", jobId, err)
		}
	})
	if err != nil {
		log.Printf("Error scheduling '%s' job: %v", jobId, err)
	}
}
