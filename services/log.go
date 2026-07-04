package services

import (
	"ego/models"
	"ego/repositories"
)

// GetAllLogs returns all broadcast logs
func GetAllLogs() ([]models.BroadcastLog, error) {
	return repositories.GetAllLogs()
}

// CreateLog creates a new broadcast log
func CreateLog(l *models.BroadcastLog) (int, error) {
	return repositories.CreateLog(l)
}

// GetTodayLogs returns today's broadcast logs
func GetTodayLogs() ([]models.BroadcastLog, error) {
	return repositories.GetTodayLogs()
}

// GetTodayBroadcastCount returns the count of today's broadcasts
func GetTodayBroadcastCount() (int, error) {
	logs, err := repositories.GetTodayLogs()
	if err != nil {
		return 0, err
	}
	return len(logs), nil
}
