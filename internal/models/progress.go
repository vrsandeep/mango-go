package models

type ProgressUpdate struct {
	JobName  string  `json:"job_name"`
	Message  string  `json:"message"`
	Progress float64 `json:"progress"`
	ItemID   int64   `json:"item_id"`
	Status   string  `json:"status"` // e.g. "in_progress", "completed", "failed"
	// Optional fields for more detailed updates
	Done bool `json:"done"`
}
