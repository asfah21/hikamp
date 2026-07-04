package services

import (
	"ego/internal/hikvision"
	"ego/models"
	"ego/repositories"
)

// GetAllSchedules returns all broadcast schedules
func GetAllSchedules() ([]models.BroadcastSchedule, error) {
	return repositories.GetAllSchedules()
}

// GetScheduleByID returns a schedule by ID
func GetScheduleByID(id int) (*models.BroadcastSchedule, error) {
	return repositories.GetScheduleByID(id)
}

// CreateSchedule creates a new schedule
func CreateSchedule(s *models.BroadcastSchedule) (int, error) {
	return repositories.CreateSchedule(s)
}

// UpdateSchedule updates an existing schedule
func UpdateSchedule(s *models.BroadcastSchedule) error {
	return repositories.UpdateSchedule(s)
}

// DeleteSchedule deletes a schedule
func DeleteSchedule(id int) error {
	return repositories.DeleteSchedule(id)
}

// SyncScheduleToDevice syncs a schedule to a Hikvision device
func SyncScheduleToDevice(scheduleID int) error {
	schedule, err := repositories.GetScheduleByID(scheduleID)
	if err != nil {
		return err
	}

	device, err := repositories.GetDeviceByID(schedule.DeviceID)
	if err != nil {
		return err
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// Build Hikvision schedule payload
	payload := buildHikvisionSchedulePayload(schedule)
	return client.CreateSchedule(payload)
}

// buildHikvisionSchedulePayload builds the Hikvision ISAPI payload from our schedule model
func buildHikvisionSchedulePayload(s *models.BroadcastSchedule) map[string]interface{} {
	payload := map[string]interface{}{
		"broadcastPlanSchemeList": []map[string]interface{}{
			{
				"planSchemeID":   s.Name,
				"enabled":        s.Enabled,
				"planSchemeName": s.Name,
				"audioOutID":     []int{1},
			},
		},
		"terminalInfoList": []map[string]interface{}{
			{
				"terminalID": 1,
				"audioOutID": []int{1},
			},
		},
	}

	// Build schedule info based on type
	scheduleList := []map[string]interface{}{
		{
			"beginTime": s.BeginTime,
			"endTime":   s.EndTime,
			"playMode":  "order",
			"operation": map[string]interface{}{
				"audioSource":   "customAudio",
				"customAudioID": []int{s.AudioID},
				"audioLevel":    5,
				"audioVolume":   s.Volume,
			},
		},
	}

	switch s.ScheduleType {
	case "daily":
		payload["broadcastPlanSchemeList"].([]map[string]interface{})[0]["dailyScheduleInfo"] = map[string]interface{}{
			"startTime":         s.BeginTime,
			"stopTime":          s.EndTime,
			"dailyScheduleList": scheduleList,
		}
	case "weekly":
		dayOfWeek := 1
		if s.DayOfWeek != nil {
			dayOfWeek = *s.DayOfWeek
		}
		payload["broadcastPlanSchemeList"].([]map[string]interface{})[0]["weeklyScheduleInfo"] = map[string]interface{}{
			"startTime": s.BeginTime,
			"stopTime":  s.EndTime,
			"weeklyScheduleList": []map[string]interface{}{
				{
					"dayOfWeek":    dayOfWeek,
					"scheduleList": scheduleList,
				},
			},
		}
	case "specific_date":
		payload["broadcastPlanSchemeList"].([]map[string]interface{})[0]["dailyScheduleInfo"] = map[string]interface{}{
			"startTime":         s.BeginTime,
			"stopTime":          s.EndTime,
			"dailyScheduleList": scheduleList,
		}
	}

	return payload
}
