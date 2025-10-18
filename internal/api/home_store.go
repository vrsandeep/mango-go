package api

import (
	"github.com/vrsandeep/mango-go/internal/models"
)

// HomeStore defines the interface for home page data operations
type HomeStore interface {
	GetContinueReading(userID int64, limit int) ([]*models.HomeSectionItem, error)
	GetNextUp(userID int64, limit int) ([]*models.HomeSectionItem, error)
	GetRecentlyAdded(limit int) ([]*models.HomeSectionItem, error)
	GetStartReading(userID int64, limit int) ([]*models.HomeSectionItem, error)
}
