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

// getStablePlanSchemeID generates a stable, unique planSchemeID for a schedule.
// Uses a fixed prefix + schedule ID + sanitized name so the ID is deterministic.
func getStablePlanSchemeID(s *models.BroadcastSchedule) string {
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
	return base
}

// SyncAllSchedulesToDevice syncs all local schedules to their respective devices.
// Each schedule becomes ONE planScheme on the device, with ALL its entries grouped
// by day_of_week in the weeklyScheduleList (7 days max). This matches how Hikvision's
// own web UI (BroadcastPlan component) handles weekly schedules.
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
	// Each schedule = ONE planScheme, not one per entry!
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

		// Build ONE planScheme for this schedule (containing ALL entries grouped by day)
		planSchemeID := getStablePlanSchemeID(&schedule)
		schemePayload := buildHikvisionSchedulePayload(&schedule, timezoneOffset, planSchemeID)

		if list, ok := schemePayload["broadcastPlanSchemeList"].([]map[string]interface{}); ok && len(list) > 0 {
			for _, dev := range schedule.Devices {
				deviceSchemes[dev.DeviceID] = append(deviceSchemes[dev.DeviceID], list[0])
				if scheduleNames[dev.DeviceID] == "" {
					scheduleNames[dev.DeviceID] = schedule.Name
				} else {
					scheduleNames[dev.DeviceID] += ", " + schedule.Name
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
		log.Printf("[SYNC ALL] Uploading %d schedule(s) [%s] to device '%s'",
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
			log.Printf("[SYNC ALL] Failed to upload %d schedule(s) to device '%s': %v", len(schemes), device.Name, err)
			fail += len(schemes)
			lastErr = err
			continue
		}
		success += len(schemes)
		log.Printf("[SYNC ALL] Synced %d schedule(s) to device '%s' (schedules: %s)", len(schemes), device.Name, scheduleNames[deviceID])
	}

	return success, fail, lastErr
}

// SyncSchedulesFromDevice fetches all plan schemes from a Hikvision device and imports them.
// Each planScheme on the device becomes ONE local schedule with entries for each day_of_week.
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

		// Determine schedule type
		scheduleType := "daily"

		// Parse all entries
		var entries []models.ScheduleEntry

		// Check for weekly schedule (weklyScheduleInfo or weeklyScheduleInfo)
		weeklyInfo, hasWeekly := scheme["weklyScheduleInfo"].(map[string]interface{})
		if !hasWeekly {
			weeklyInfo, hasWeekly = scheme["weeklyScheduleInfo"].(map[string]interface{})
		}

		if hasWeekly {
			scheduleType = "weekly"

			if wList, ok := weeklyInfo["weeklyScheduleList"].([]interface{}); ok {
				for _, wItem := range wList {
					wEntry, ok := wItem.(map[string]interface{})
					if !ok {
						continue
					}

					var dayOfWeek int
					if d, ok := wEntry["dayOfWeek"].(float64); ok {
						dayOfWeek = int(d)
					}

					if schedList, ok := wEntry["scheduleList"].([]interface{}); ok {
						for _, schedItem := range schedList {
							schedEntry, ok := schedItem.(map[string]interface{})
							if !ok {
								continue
							}

							beginTime, _ := schedEntry["beginTime"].(string)
							endTime, _ := schedEntry["endTime"].(string)

							// Clean timezone suffix
							re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
							beginTime = re.ReplaceAllString(beginTime, "")
							endTime = re.ReplaceAllString(endTime, "")

							volume := 50
							if op, ok := schedEntry["operation"].(map[string]interface{}); ok {
								if v, ok := op["audioVolume"].(float64); ok {
									volume = int(v)
								}
							}

							dow := dayOfWeek
							entries = append(entries, models.ScheduleEntry{
								BeginTime: beginTime,
								EndTime:   endTime,
								Volume:    volume,
								DayOfWeek: &dow,
							})
						}
					}
				}
			}
		}

		// Check for daily schedule
		if dailyInfo, ok := scheme["dailyScheduleInfo"].(map[string]interface{}); !hasWeekly && ok {
			if schedList, ok := dailyInfo["dailyScheduleList"].([]interface{}); ok {
				for _, schedItem := range schedList {
					entry, ok := schedItem.(map[string]interface{})
					if !ok {
						continue
					}

					beginTime, _ := entry["beginTime"].(string)
					endTime, _ := entry["endTime"].(string)

					// Clean timezone suffix
					re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
					beginTime = re.ReplaceAllString(beginTime, "")
					endTime = re.ReplaceAllString(endTime, "")

					volume := 50
					if op, ok := entry["operation"].(map[string]interface{}); ok {
						if v, ok := op["audioVolume"].(float64); ok {
							volume = int(v)
						}
					}

					entries = append(entries, models.ScheduleEntry{
						BeginTime: beginTime,
						EndTime:   endTime,
						Volume:    volume,
						DayOfWeek: nil, // daily = no day_of_week
					})
				}
			}
		}

		// Also try dailyscheduleInfo if dailyScheduleInfo wasn't found
		if dailyInfo, ok := scheme["dailyscheduleInfo"].(map[string]interface{}); !hasWeekly && ok {
			if schedList, ok := dailyInfo["dailyScheduleList"].([]interface{}); ok {
				for _, schedItem := range schedList {
					entry, ok := schedItem.(map[string]interface{})
					if !ok {
						continue
					}

					beginTime, _ := entry["beginTime"].(string)
					endTime, _ := entry["endTime"].(string)

					re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
					beginTime = re.ReplaceAllString(beginTime, "")
					endTime = re.ReplaceAllString(endTime, "")

					volume := 50
					if op, ok := entry["operation"].(map[string]interface{}); ok {
						if v, ok := op["audioVolume"].(float64); ok {
							volume = int(v)
						}
					}

					entries = append(entries, models.ScheduleEntry{
						BeginTime: beginTime,
						EndTime:   endTime,
						Volume:    volume,
						DayOfWeek: nil,
					})
				}
			}
		}

		if len(entries) == 0 {
			log.Printf("[SYNC FROM DEVICE] Skipping planScheme '%s': no parseable entries", planSchemeID)
			continue
		}

		name := planSchemeName
		if name == "" {
			name = planSchemeID
		}
		if name == "" {
			name = fmt.Sprintf("Imported Schedule %d", synced+1)
		}

		// Create schedule with all entries and one device
		schedule := &models.BroadcastSchedule{
			Name:         name,
			ScheduleType: scheduleType,
			Enabled:      enabled,
			Entries:      entries,
			Devices: []models.ScheduleDevice{
				{
					DeviceID: deviceID,
				},
			},
		}

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

// lookupAudioHikvisionID looks up the Hikvision audio ID for a given audio file ID.
// Returns 0 if not found.
func lookupAudioHikvisionID(audioID int) int {
	if audioID <= 0 {
		return 0
	}
	audioFile, err := repositories.GetAudioFileByID(audioID)
	if err != nil || audioFile == nil {
		return 0
	}
	if audioFile.HikvisionAudioID == nil || *audioFile.HikvisionAudioID == 0 {
		return 0
	}
	return *audioFile.HikvisionAudioID
}

// buildHikvisionSchedulePayload builds the Hikvision ISAPI payload from a schedule.
// All entries are grouped by day_of_week into a single planScheme.
// This matches the structure used by Hikvision's own BroadcastPlan Vue component (core-2.js).
func buildHikvisionSchedulePayload(s *models.BroadcastSchedule, timezoneOffset string, planSchemeID string) map[string]interface{} {
	now := time.Now()

	// Use schedule's StartDate/EndDate when set, otherwise fall back to defaults
	startDateStr := now.Format("2006-01-02") + "+" + timezoneOffset
	if s.StartDate != nil && *s.StartDate != "" {
		startDateStr = *s.StartDate + "+" + timezoneOffset
	}
	stopDateStr := now.AddDate(0, 0, 7).Format("2006-01-02") + "+" + timezoneOffset
	if s.EndDate != nil && *s.EndDate != "" {
		stopDateStr = *s.EndDate + "+" + timezoneOffset
	}

	planScheme := map[string]interface{}{
		"planSchemeID":   planSchemeID,
		"planSchemeName": s.Name,
		"enabled":        s.Enabled,
		"audioOutID":     []int{1},
	}

	switch s.ScheduleType {
	case "daily":
		// Daily schedule: all entries go into a single day (no day_of_week)
		var scheduleEntries []map[string]interface{}
		for _, entry := range s.Entries {
			// Skip entries with specific day_of_week for daily schedule
			if entry.DayOfWeek != nil {
				continue
			}

			beginTime := formatTimeForHikvision(entry.BeginTime, timezoneOffset)
			endTime := formatTimeForHikvision(entry.EndTime, timezoneOffset)
			audioID := lookupAudioHikvisionID(entry.AudioID)

			scheduleEntry := map[string]interface{}{
				"beginTime": beginTime,
				"endTime":   endTime,
				"playMode":  "order",
				"operation": map[string]interface{}{
					"audioSource":   "customAudio",
					"customAudioID": []int{audioID},
					"audioLevel":    5,
					"audioVolume":   entry.Volume,
				},
			}
			scheduleEntries = append(scheduleEntries, scheduleEntry)
		}

		if len(scheduleEntries) > 0 {
			planScheme["dailyScheduleInfo"] = map[string]interface{}{
				"startTime":         startDateStr,
				"stopTime":          stopDateStr,
				"dailyScheduleList": scheduleEntries,
			}
		}

	case "weekly":
		// Weekly schedule: group entries by day_of_week (1=Monday ... 7=Sunday)
		// Create a map of day_of_week -> entries
		dayEntries := map[int][]map[string]interface{}{}

		for _, entry := range s.Entries {
			dayOfWeek := 1 // default Monday
			if entry.DayOfWeek != nil {
				dayOfWeek = *entry.DayOfWeek
			}

			beginTime := formatTimeForHikvision(entry.BeginTime, timezoneOffset)
			endTime := formatTimeForHikvision(entry.EndTime, timezoneOffset)
			audioID := lookupAudioHikvisionID(entry.AudioID)

			scheduleEntry := map[string]interface{}{
				"beginTime": beginTime,
				"endTime":   endTime,
				"playMode":  "order",
				"operation": map[string]interface{}{
					"audioSource":   "customAudio",
					"customAudioID": []int{audioID},
					"audioLevel":    5,
					"audioVolume":   entry.Volume,
				},
			}
			dayEntries[dayOfWeek] = append(dayEntries[dayOfWeek], scheduleEntry)
		}

		if len(dayEntries) > 0 {
			// Build weeklyScheduleList with entries grouped by day
			var weeklyScheduleList []map[string]interface{}
			for day := 1; day <= 7; day++ {
				entries, ok := dayEntries[day]
				if !ok || len(entries) == 0 {
					continue
				}
				weeklyScheduleList = append(weeklyScheduleList, map[string]interface{}{
					"dayOfWeek":    day,
					"scheduleList": entries,
				})
			}

			if len(weeklyScheduleList) > 0 {
				planScheme["weklyScheduleInfo"] = map[string]interface{}{
					"startTime":          startDateStr,
					"stopTime":           stopDateStr,
					"weeklyScheduleList": weeklyScheduleList,
				}
			}
		}

	case "specific_date":
		dateStr := startDateStr
		if s.SpecificDate != nil && *s.SpecificDate != "" {
			dateStr = *s.SpecificDate
		}

		var scheduleEntries []map[string]interface{}
		for _, entry := range s.Entries {
			beginTime := formatTimeForHikvision(entry.BeginTime, timezoneOffset)
			endTime := formatTimeForHikvision(entry.EndTime, timezoneOffset)
			audioID := lookupAudioHikvisionID(entry.AudioID)

			scheduleEntry := map[string]interface{}{
				"beginTime": beginTime,
				"endTime":   endTime,
				"playMode":  "order",
				"operation": map[string]interface{}{
					"audioSource":   "customAudio",
					"customAudioID": []int{audioID},
					"audioLevel":    5,
					"audioVolume":   entry.Volume,
				},
			}
			scheduleEntries = append(scheduleEntries, scheduleEntry)
		}

		if len(scheduleEntries) > 0 {
			planScheme["dailyScheduleInfo"] = map[string]interface{}{
				"startTime":         dateStr,
				"stopTime":          dateStr,
				"dailyScheduleList": scheduleEntries,
			}
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
