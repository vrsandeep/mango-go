package models

type ProgressUpdate struct {
	JobID    string  `json:"jobId"`
	Message  string  `json:"message"`
	Progress float64 `json:"progress"`
	ItemID   int64   `json:"item_id"`
	Status   string  `json:"status"` // e.g. "in_progress", "completed", "failed"
	// Optional fields for more detailed updates
	Done bool `json:"done"`
}
