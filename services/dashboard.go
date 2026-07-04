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

	// Device stats
	devices, err := repositories.GetAllDevices()
	if err != nil {
		return nil, err
	}
	data.TotalDevices = len(devices)
	for _, d := range devices {
		if d.Status == "online" {
			data.OnlineDevices++
		} else {
			data.OfflineDevices++
		}
	}

	// Audio stats
	audioFiles, err := repositories.GetAllAudioFiles()
	if err != nil {
		return nil, err
	}
	data.TotalAudioFiles = len(audioFiles)

	// Today's broadcast count
	logs, err := repositories.GetTodayLogs()
	if err != nil {
		return nil, err
	}
	data.TodayBroadcastCount = len(logs)

	// Recent logs (last 5)
	if len(logs) > 5 {
		logs = logs[:5]
	}
	data.RecentLogs = logs

	// Next schedule
	schedules, err := repositories.GetAllSchedules()
	if err != nil {
		return nil, err
	}
	for _, s := range schedules {
		if s.Enabled {
			data.NextSchedule = &s
			break
		}
	}

	return data, nil
}
