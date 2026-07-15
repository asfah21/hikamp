package services

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

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

// getStablePlanSchemeID generates a stable, unique planSchemeID for a schedule+entry combination.
func getStablePlanSchemeID(s *models.BroadcastSchedule, entryIdx int) string {
	safeName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ' {
			return r
		}
		return '_'
	}, s.Name)

	base := fmt.Sprintf("sch_%s", safeName)
	if s.ID > 0 {
		base = fmt.Sprintf("sch_%d_%s", s.ID, safeName)
	}
	if entryIdx > 0 {
		base = fmt.Sprintf("%s_e%d", base, entryIdx)
	}
	return base
}

// SyncAllSchedulesToDevice syncs all local schedules to their respective devices.
// Aggregates ALL entries from ALL schedules per-device, then sends them in a single
// AddPlanScheme request per device. This avoids the destructive overwrite issue where
// sending per-schedule would erase previously synced schedules.
func SyncAllSchedulesToDevice() (int, int, error) {
	schedules, err := repositories.GetAllSchedules()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get schedules: %w", err)
	}

	// Get timezone offset
	timezoneOffset := "08:00"
	location, err := repositories.GetPrayerLocation()
	if err == nil && location.Timezone != "" {
		timezoneOffset = getTimezoneOffset(location.Timezone)
	}

	// Build per-device scheme lists from ALL enabled schedules
	deviceSchemes := map[int][]map[string]interface{}{}
	scheduleNames := map[int]string{} // deviceID -> schedule names for logging

	for _, schedule := range schedules {
		if !schedule.Enabled {
			log.Printf("[SYNC ALL] Skipping disabled schedule '%s'", schedule.Name)
			continue
		}
		if len(schedule.Entries) == 0 {
			log.Printf("[SYNC ALL] Skipping schedule '%s': no entries", schedule.Name)
			continue
		}
		if len(schedule.Devices) == 0 {
			log.Printf("[SYNC ALL] Skipping schedule '%s': no target devices", schedule.Name)
			continue
		}

		for _, dev := range schedule.Devices {
			for idx, entry := range schedule.Entries {
				if entry.BeginTime == "" || entry.EndTime == "" {
					log.Printf("[SYNC ALL] Skipping entry %d of schedule '%s': begin/end time empty", idx, schedule.Name)
					continue
				}

				audioFile, err := repositories.GetAudioFileByID(entry.AudioID)
				if err != nil {
					log.Printf("[SYNC ALL] Skipping entry %d of '%s': audio file not found: %v", idx, schedule.Name, err)
					continue
				}
				if audioFile.HikvisionAudioID == nil || *audioFile.HikvisionAudioID == 0 {
					log.Printf("[SYNC ALL] Skipping entry %d of '%s': audio '%s' has no Hikvision audio ID", idx, schedule.Name, audioFile.Name)
					continue
				}

				planSchemeID := getStablePlanSchemeID(&schedule, idx)
				schemePayload := buildHikvisionSchedulePayload(&schedule, &entry, timezoneOffset, planSchemeID, *audioFile.HikvisionAudioID)

				if list, ok := schemePayload["broadcastPlanSchemeList"].([]map[string]interface{}); ok && len(list) > 0 {
					deviceSchemes[dev.DeviceID] = append(deviceSchemes[dev.DeviceID], list[0])
					if scheduleNames[dev.DeviceID] == "" {
						scheduleNames[dev.DeviceID] = schedule.Name
					} else {
						scheduleNames[dev.DeviceID] += ", " + schedule.Name
					}
				}
			}
		}
	}

	if len(deviceSchemes) == 0 {
		return 0, 0, fmt.Errorf("no valid schedules to sync")
	}

	success := 0
	fail := 0
	var lastErr error

	// For each device, clear existing schemes and upload ALL schemes in one request
	for deviceID, schemes := range deviceSchemes {
		device, err := repositories.GetDeviceByID(deviceID)
		if err != nil {
			log.Printf("[SYNC ALL] Device ID %d not found: %v", deviceID, err)
			fail += len(schemes)
			lastErr = err
			continue
		}

		client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

		// Search existing schemes to clear them
		existingSchemes := []map[string]interface{}{}
		searchResult, searchErr := client.SearchPlanScheme()
		if searchErr == nil {
			if schemesMap, ok := searchResult.(map[string]interface{}); ok {
				if list, ok := schemesMap["broadcastPlanSchemeList"].([]interface{}); ok {
					for _, item := range list {
						if scheme, ok := item.(map[string]interface{}); ok {
							existingSchemes = append(existingSchemes, scheme)
						}
					}
				}
			}
		} else {
			log.Printf("[SYNC ALL] SearchPlanScheme failed on device '%s': %v", device.Name, searchErr)
		}

		// Clear existing schemes on device
		if len(existingSchemes) > 0 {
			log.Printf("[SYNC ALL] Removing %d existing schedule(s) from device '%s'", len(existingSchemes), device.Name)
			clearPayload := map[string]interface{}{
				"broadcastPlanSchemeList": []map[string]interface{}{},
				"terminalInfoList": []map[string]interface{}{
					{
						"terminalID": 1,
						"audioOutID": []int{1},
					},
				},
			}
			clearErr := client.ModifyPlanScheme(clearPayload)
			if clearErr != nil {
				log.Printf("[SYNC ALL] ModifyPlanScheme clear failed on '%s': %v — trying AddPlanScheme", device.Name, clearErr)
				clearErr = client.CreateSchedule(clearPayload)
			}
			if clearErr != nil {
				log.Printf("[SYNC ALL] Failed to clear schedules on device '%s': %v", device.Name, clearErr)
			}
			time.Sleep(500 * time.Millisecond)
		}

		// Upload ALL schemes from ALL schedules in ONE AddPlanScheme call
		log.Printf("[SYNC ALL] Uploading %d entries for schedules [%s] to device '%s'",
			len(schemes), scheduleNames[deviceID], device.Name)

		addPayload := map[string]interface{}{
			"broadcastPlanSchemeList": schemes,
			"terminalInfoList": []map[string]interface{}{
				{
					"terminalID": 1,
					"audioOutID": []int{1},
				},
			},
		}
		err = client.CreateSchedule(addPayload)
		if err != nil {
			log.Printf("[SYNC ALL] Failed to upload %d entries to device '%s': %v", len(schemes), device.Name, err)
			fail += len(schemes)
			lastErr = err
			continue
		}
		success += len(schemes)
		log.Printf("[SYNC ALL] Synced %d entries to device '%s' (schedules: %s)", len(schemes), device.Name, scheduleNames[deviceID])
	}

	return success, fail, lastErr
}

// SyncSchedulesFromDevice fetches all plan schemes from a Hikvision device and imports them.
func SyncSchedulesFromDevice(deviceID int) (int, error) {
	device, err := repositories.GetDeviceByID(deviceID)
	if err != nil {
		return 0, fmt.Errorf("device not found: %w", err)
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	schemes, err := client.SearchPlanScheme()
	if err != nil {
		return 0, fmt.Errorf("sync from device failed: this device does not support searching schedules: %w", err)
	}

	schemesMap, ok := schemes.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected response format from SearchPlanScheme")
	}

	list, ok := schemesMap["broadcastPlanSchemeList"].([]interface{})
	if !ok || len(list) == 0 {
		return 0, nil
	}

	// Delete all existing local schedules that reference this device
	if err := repositories.DeleteSchedulesByDevice(deviceID); err != nil {
		return 0, fmt.Errorf("failed to clear local schedules for device: %w", err)
	}

	synced := 0
	for _, item := range list {
		scheme, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		planSchemeID, _ := scheme["planSchemeID"].(string)
		planSchemeName, _ := scheme["planSchemeName"].(string)
		enabled, _ := scheme["enabled"].(bool)

		var beginTime, endTime string
		var scheduleType string = "daily"
		var dayOfWeek int

		if dailyInfo, ok := scheme["dailyscheduleInfo"].(map[string]interface{}); ok {
			if schedList, ok := dailyInfo["dailyScheduleList"].([]interface{}); ok && len(schedList) > 0 {
				if entry, ok := schedList[0].(map[string]interface{}); ok {
					beginTime, _ = entry["beginTime"].(string)
					endTime, _ = entry["endTime"].(string)
				}
			}
		}

		if weeklyInfo, ok := scheme["weklyScheduleInfo"].(map[string]interface{}); ok {
			scheduleType = "weekly"
			if wList, ok := weeklyInfo["weeklyScheduleList"].([]interface{}); ok && len(wList) > 0 {
				if wEntry, ok := wList[0].(map[string]interface{}); ok {
					if d, ok := wEntry["dayOfWeek"].(float64); ok {
						dayOfWeek = int(d)
					}
					if schedList, ok := wEntry["scheduleList"].([]interface{}); ok && len(schedList) > 0 {
						if schedEntry, ok := schedList[0].(map[string]interface{}); ok {
							beginTime, _ = schedEntry["beginTime"].(string)
							endTime, _ = schedEntry["endTime"].(string)
						}
					}
				}
			}
		} else if weeklyInfo, ok := scheme["weeklyScheduleInfo"].(map[string]interface{}); ok {
			scheduleType = "weekly"
			if wList, ok := weeklyInfo["weeklyScheduleList"].([]interface{}); ok && len(wList) > 0 {
				if wEntry, ok := wList[0].(map[string]interface{}); ok {
					if d, ok := wEntry["dayOfWeek"].(float64); ok {
						dayOfWeek = int(d)
					}
					if schedList, ok := wEntry["scheduleList"].([]interface{}); ok && len(schedList) > 0 {
						if schedEntry, ok := schedList[0].(map[string]interface{}); ok {
							beginTime, _ = schedEntry["beginTime"].(string)
							endTime, _ = schedEntry["endTime"].(string)
						}
					}
				}
			}
		}

		// Clean timezone suffix
		re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
		beginTime = re.ReplaceAllString(beginTime, "")
		endTime = re.ReplaceAllString(endTime, "")

		name := planSchemeName
		if name == "" {
			name = planSchemeID
		}
		if name == "" {
			name = fmt.Sprintf("Imported Schedule %d", synced+1)
		}

		// Create schedule with one entry and one device
		schedule := &models.BroadcastSchedule{
			Name:         name,
			ScheduleType: scheduleType,
			Enabled:      enabled,
			Entries: []models.ScheduleEntry{
				{
					BeginTime: beginTime,
					EndTime:   endTime,
					Volume:    50,
				},
			},
			Devices: []models.ScheduleDevice{
				{
					DeviceID: deviceID,
				},
			},
		}

		if scheduleType == "weekly" && dayOfWeek > 0 {
			schedule.DayOfWeek = &dayOfWeek
		}

		// Try to find matching audio by name/planSchemeID
		// For imported schedules we leave AudioID as 0 (unlinked) — user can edit later

		_, err := repositories.CreateSchedule(schedule)
		if err != nil {
			log.Printf("[SYNC FROM DEVICE] Failed to create schedule '%s': %v", name, err)
			continue
		}
		synced++
	}

	return synced, nil
}

// getTimezoneOffset converts a timezone name to offset string (e.g., "08:00").
func getTimezoneOffset(tzName string) string {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return "08:00"
	}
	_, offset := time.Now().In(loc).Zone()
	hours := offset / 3600
	mins := (offset % 3600) / 60
	if hours >= 0 {
		return fmt.Sprintf("%02d:%02d", hours, mins)
	}
	return fmt.Sprintf("-%02d:%02d", -hours, mins)
}

// formatTimeForHikvision converts a time string to "HH:MM:SS+HH:MM" format.
func formatTimeForHikvision(timeStr string, timezoneOffset string) string {
	if timeStr == "" {
		return ""
	}

	re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
	timeStr = re.ReplaceAllString(timeStr, "")

	reSpace := regexp.MustCompile(` \d{2}:\d{2}$`)
	timeStr = reSpace.ReplaceAllString(timeStr, "")

	parts := strings.Split(timeStr, ":")
	switch len(parts) {
	case 2:
		return fmt.Sprintf("%s:%s:00+%s", parts[0], parts[1], timezoneOffset)
	case 3:
		return fmt.Sprintf("%s:%s:%s+%s", parts[0], parts[1], parts[2], timezoneOffset)
	default:
		return timeStr + "+" + timezoneOffset
	}
}

// buildHikvisionSchedulePayload builds the Hikvision ISAPI payload from a schedule + entry.
func buildHikvisionSchedulePayload(s *models.BroadcastSchedule, entry *models.ScheduleEntry, timezoneOffset string, planSchemeID string, hikvisionAudioID int) map[string]interface{} {
	beginTime := formatTimeForHikvision(entry.BeginTime, timezoneOffset)
	endTime := formatTimeForHikvision(entry.EndTime, timezoneOffset)

	now := time.Now()
	today := now.Format("2006-01-02") + "+" + timezoneOffset
	futureDate := now.AddDate(0, 0, 7).Format("2006-01-02") + "+" + timezoneOffset

	scheduleEntry := map[string]interface{}{
		"beginTime": beginTime,
		"endTime":   endTime,
		"playMode":  "order",
		"operation": map[string]interface{}{
			"audioSource":   "customAudio",
			"customAudioID": []int{hikvisionAudioID},
			"audioLevel":    5,
			"audioVolume":   entry.Volume,
		},
	}

	planScheme := map[string]interface{}{
		"planSchemeID":   planSchemeID,
		"planSchemeName": s.Name,
		"enabled":        s.Enabled,
		"audioOutID":     []int{1},
	}

	switch s.ScheduleType {
	case "daily":
		planScheme["dailyScheduleInfo"] = map[string]interface{}{
			"startTime": today,
			"stopTime":  futureDate,
			"dailyScheduleList": []map[string]interface{}{
				scheduleEntry,
			},
		}
	case "weekly":
		dayOfWeek := 1
		if s.DayOfWeek != nil {
			dayOfWeek = *s.DayOfWeek
		}
		planScheme["weeklyScheduleInfo"] = map[string]interface{}{
			"startTime": today,
			"stopTime":  futureDate,
			"weeklyScheduleList": []map[string]interface{}{
				{
					"dayOfWeek":    dayOfWeek,
					"scheduleList": []map[string]interface{}{scheduleEntry},
				},
			},
		}
	case "specific_date":
		dateStr := today
		if s.SpecificDate != nil && *s.SpecificDate != "" {
			dateStr = *s.SpecificDate
		}
		planScheme["dailyScheduleInfo"] = map[string]interface{}{
			"startTime": dateStr,
			"stopTime":  dateStr,
			"dailyScheduleList": []map[string]interface{}{
				scheduleEntry,
			},
		}
	}

	payload := map[string]interface{}{
		"broadcastPlanSchemeList": []map[string]interface{}{
			planScheme,
		},
		"terminalInfoList": []map[string]interface{}{
			{
				"terminalID": 1,
				"audioOutID": []int{1},
			},
		},
	}

	return payload
}
