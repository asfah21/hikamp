package services

import (
	"ego/models"
	"ego/repositories"
)

// DashboardData holds all dashboard statistics
type DashboardData struct {
	TotalDevices        int
	OnlineDevices       int
	OfflineDevices      int
	TotalAudioFiles     int
	TodayBroadcastCount int
	NextSchedule        *models.BroadcastSchedule
	RecentLogs          []models.BroadcastLog
}

// GetDashboardData gathers all dashboard statistics
func GetDashboardData() (*DashboardData, error) {
	data := &DashboardData{}

	// Device stats (graceful fallback)
	if devices, err := repositories.GetAllDevices(); err == nil {
		data.TotalDevices = len(devices)
		for _, d := range devices {
			if d.Status == "online" {
				data.OnlineDevices++
			} else {
				data.OfflineDevices++
			}
		}
	}

	// Audio stats (graceful fallback)
	if audioFiles, err := repositories.GetAllAudioFiles(); err == nil {
		data.TotalAudioFiles = len(audioFiles)
	}

	// Today's broadcast count (graceful fallback)
	if logs, err := repositories.GetTodayLogs(); err == nil {
		data.TodayBroadcastCount = len(logs)
		// Recent logs (last 5)
		if len(logs) > 5 {
			logs = logs[:5]
		}
		data.RecentLogs = logs
	}

	// Next schedule (graceful fallback)
	if schedules, err := repositories.GetAllSchedules(); err == nil {
		for _, s := range schedules {
			if s.Enabled {
				data.NextSchedule = &s
				break
			}
		}
	}

	return data, nil
}
