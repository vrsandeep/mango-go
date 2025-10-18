package api

import (
	"github.com/stretchr/testify/mock"
	"github.com/vrsandeep/mango-go/internal/models"
)

// MockHomeStore is a mock implementation of HomeStore interface
type MockHomeStore struct {
	mock.Mock
}

// GetContinueReading mocks the GetContinueReading method
func (m *MockHomeStore) GetContinueReading(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]*models.HomeSectionItem), args.Error(1)
}

// GetNextUp mocks the GetNextUp method
func (m *MockHomeStore) GetNextUp(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]*models.HomeSectionItem), args.Error(1)
}

// GetRecentlyAdded mocks the GetRecentlyAdded method
func (m *MockHomeStore) GetRecentlyAdded(limit int) ([]*models.HomeSectionItem, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.HomeSectionItem), args.Error(1)
}

// GetStartReading mocks the GetStartReading method
func (m *MockHomeStore) GetStartReading(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]*models.HomeSectionItem), args.Error(1)
}
