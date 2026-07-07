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
// Uses schedule ID + name to ensure it's stable across syncs (no random/timestamp component).
func getStablePlanSchemeID(s *models.BroadcastSchedule) string {
	// Sanitize name to remove characters that might cause issues in planSchemeID
	safeName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ' {
			return r
		}
		return '_'
	}, s.Name)

	if s.ID > 0 {
		return fmt.Sprintf("sch_%d_%s", s.ID, safeName)
	}
	return fmt.Sprintf("sch_%s", safeName)
}

// SyncScheduleToDevice syncs a schedule to a Hikvision device.
// Uses DeletePlanScheme + AddPlanScheme pattern for reliability.
// DeletePlanScheme removes the existing schedule with the same planSchemeID,
// then AddPlanScheme creates the new one. This avoids 403 "Invalid Operation"
// errors from ModifyPlanScheme on some firmware versions.
// Also handles devices that don't support SearchPlanScheme (GET 403 methodNotAllowed).
// Uses HikvisionAudioID from the audio_files table as customAudioID in the payload.
func SyncScheduleToDevice(scheduleID int) error {
	schedule, err := repositories.GetScheduleByID(scheduleID)
	if err != nil {
		return err
	}

	// Validate begin_time and end_time are not empty
	if schedule.BeginTime == "" || schedule.EndTime == "" {
		return fmt.Errorf("cannot sync schedule '%s': begin_time and end_time must be set. Edit the schedule and set the time values first", schedule.Name)
	}

	device, err := repositories.GetDeviceByID(schedule.DeviceID)
	if err != nil {
		return err
	}

	// Lookup the audio file to get HikvisionAudioID
	audioFile, err := repositories.GetAudioFileByID(schedule.AudioID)
	if err != nil {
		return fmt.Errorf("audio file not found (ID: %d): %w", schedule.AudioID, err)
	}
	if audioFile.HikvisionAudioID == nil || *audioFile.HikvisionAudioID == 0 {
		return fmt.Errorf("audio file '%s' has no Hikvision audio ID. Upload the audio to the device first", audioFile.Name)
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// Get timezone offset from location settings
	timezoneOffset := "08:00" // default fallback
	location, err := repositories.GetPrayerLocation()
	if err == nil && location.Timezone != "" {
		timezoneOffset = getTimezoneOffset(location.Timezone)
	}

	// Generate a stable planSchemeID
	planSchemeID := getStablePlanSchemeID(schedule)

	log.Printf("[SYNC] planSchemeID=%s, beginTime=%s, endTime=%s, hikvisionAudioID=%d", planSchemeID, schedule.BeginTime, schedule.EndTime, *audioFile.HikvisionAudioID)

	// Step 1: Try to delete existing plan scheme with the same ID (ignore error if not found)
	// This handles both:
	//   - Devices that support SearchPlanScheme: we can't search, so we just try delete
	//   - Devices that don't support SearchPlanScheme: delete may also fail, we ignore
	log.Printf("[SYNC] Attempting to remove existing planScheme '%s' if it exists", planSchemeID)
	delErr := client.DeletePlanScheme(planSchemeID)
	if delErr != nil {
		// Delete might fail if scheme doesn't exist or device doesn't support ModifyPlanScheme for delete
		// Either way, we continue to try Add
		log.Printf("[SYNC] Delete existing scheme (if any) result: %v (continuing)", delErr)
	}
	// Small delay to let device process the deletion
	time.Sleep(500 * time.Millisecond)

	// Step 2: Build and send the schedule payload using AddPlanScheme
	payload := buildHikvisionSchedulePayload(schedule, timezoneOffset, planSchemeID, *audioFile.HikvisionAudioID)

	err = client.CreateSchedule(payload)
	if err != nil {
		return fmt.Errorf("sync schedule failed: %w", err)
	}

	log.Printf("[SYNC] Schedule synced successfully: %s", planSchemeID)
	return nil
}

// SyncAllSchedulesToDevice syncs all local schedules to their respective devices.
// Returns a summary of successes and failures.
func SyncAllSchedulesToDevice() (int, int, error) {
	schedules, err := repositories.GetAllSchedules()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get schedules: %w", err)
	}

	success := 0
	fail := 0
	var lastErr error

	for _, s := range schedules {
		err := SyncScheduleToDevice(s.ID)
		if err != nil {
			log.Printf("[SYNC ALL] Failed to sync schedule '%s' (ID: %d): %v", s.Name, s.ID, err)
			fail++
			lastErr = err
		} else {
			success++
		}
	}

	return success, fail, lastErr
}

// SyncSchedulesFromDevice fetches all plan schemes from a Hikvision device
// and creates/updates local schedule records.
// Note: Some firmware versions do not support SearchPlanScheme (GET) and return
// 403 "methodNotAllowed". In that case, sync-from-device is not possible.
func SyncSchedulesFromDevice(deviceID int) (int, error) {
	device, err := repositories.GetDeviceByID(deviceID)
	if err != nil {
		return 0, fmt.Errorf("device not found: %w", err)
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// Search for all plan schemes on the device
	// Note: Some firmware versions return 403 "methodNotAllowed" for this endpoint.
	schemes, err := client.SearchPlanScheme()
	if err != nil {
		return 0, fmt.Errorf("sync from device failed: this device does not support searching schedules (SearchPlanScheme returned 403). Try creating schedules locally and syncing to device instead: %w", err)
	}

	// Parse the response
	schemesMap, ok := schemes.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected response format from SearchPlanScheme")
	}

	list, ok := schemesMap["broadcastPlanSchemeList"].([]interface{})
	if !ok || len(list) == 0 {
		return 0, nil // no schedules on device
	}

	synced := 0
	for _, item := range list {
		scheme, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		planSchemeID, _ := scheme["planSchemeID"].(string)
		enabled, _ := scheme["enabled"].(bool)

		// Try to extract schedule info from dailyscheduleInfo or weeklyScheduleInfo
		var beginTime, endTime string
		var scheduleType string = "daily"

		if dailyInfo, ok := scheme["dailyscheduleInfo"].(map[string]interface{}); ok {
			if list, ok := dailyInfo["dailyScheduleList"].([]interface{}); ok && len(list) > 0 {
				if entry, ok := list[0].(map[string]interface{}); ok {
					beginTime, _ = entry["beginTime"].(string)
					endTime, _ = entry["endTime"].(string)
				}
			}
		} else if weeklyInfo, ok := scheme["weeklyScheduleInfo"].(map[string]interface{}); ok {
			scheduleType = "weekly"
			if list, ok := weeklyInfo["weeklyScheduleList"].([]interface{}); ok && len(list) > 0 {
				if entry, ok := list[0].(map[string]interface{}); ok {
					if schedList, ok := entry["scheduleList"].([]interface{}); ok && len(schedList) > 0 {
						if schedEntry, ok := schedList[0].(map[string]interface{}); ok {
							beginTime, _ = schedEntry["beginTime"].(string)
							endTime, _ = schedEntry["endTime"].(string)
						}
					}
				}
			}
		}

		// Clean timezone suffix from times
		re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
		beginTime = re.ReplaceAllString(beginTime, "")
		endTime = re.ReplaceAllString(endTime, "")

		// Generate a name from the planSchemeID
		name := planSchemeID
		if name == "" {
			name = fmt.Sprintf("Imported Schedule %d", synced+1)
		}

		// Create a local schedule record
		schedule := &models.BroadcastSchedule{
			Name:         name,
			DeviceID:     deviceID,
			ScheduleType: scheduleType,
			BeginTime:    beginTime,
			EndTime:      endTime,
			Volume:       50,
			Enabled:      enabled,
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

// getTimezoneOffset converts a timezone name (e.g., "Asia/Jakarta") to offset string (e.g., "08:00").
// Note: Returns WITHOUT the "+" prefix because Web UI uses "+" as separator:
// "HH:MM:SS+08:00" — the "+" is added by the caller.
// For negative offsets, the sign IS included (e.g., "-05:00").
func getTimezoneOffset(tzName string) string {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return "08:00"
	}
	_, offset := time.Now().In(loc).Zone()
	hours := offset / 3600
	mins := (offset % 3600) / 60
	if hours >= 0 {
		// Positive offset: return without "+" prefix (caller adds "+" separator)
		return fmt.Sprintf("%02d:%02d", hours, mins)
	}
	// Negative offset: include "-" prefix
	return fmt.Sprintf("-%02d:%02d", -hours, mins)
}

// formatTimeForHikvision converts a time string to "HH:MM:SS+HH:MM" format.
// Hikvision Web UI uses "+" separator between time and timezone (e.g., "22:02:00+08:00").
// Handles various input formats:
//   - "HH:MM" -> "HH:MM:SS+HH:MM"
//   - "HH:MM:SS" -> "HH:MM:SS+HH:MM"
//   - "HH:MM:SS+08:00" -> "HH:MM:SS+HH:MM" (timezone is replaced)
//   - "" -> returns empty string (caller should validate)
func formatTimeForHikvision(timeStr string, timezoneOffset string) string {
	if timeStr == "" {
		return ""
	}

	// Remove any existing timezone suffix (match any +/-HH:MM pattern at the end)
	re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
	timeStr = re.ReplaceAllString(timeStr, "")

	// Also remove any existing timezone with space separator (e.g., " 08:00" at the end)
	reSpace := regexp.MustCompile(` \d{2}:\d{2}$`)
	timeStr = reSpace.ReplaceAllString(timeStr, "")

	// Count colons to determine format
	parts := strings.Split(timeStr, ":")
	switch len(parts) {
	case 2:
		// HH:MM -> HH:MM:SS+HH:MM (plus separator)
		return fmt.Sprintf("%s:%s:00+%s", parts[0], parts[1], timezoneOffset)
	case 3:
		// HH:MM:SS -> HH:MM:SS+HH:MM (plus separator)
		return fmt.Sprintf("%s:%s:%s+%s", parts[0], parts[1], parts[2], timezoneOffset)
	default:
		// Unknown format, return as-is with timezone (plus separator)
		return timeStr + "+" + timezoneOffset
	}
}

// buildHikvisionSchedulePayload builds the Hikvision ISAPI payload from our schedule model.
// Uses map[string]interface{} payload matching the official Hikvision Web UI format.
// hikvisionAudioID is the customAudioID from the Hikvision device (not the local audio_files.id).
func buildHikvisionSchedulePayload(s *models.BroadcastSchedule, timezoneOffset string, planSchemeID string, hikvisionAudioID int) map[string]interface{} {
	// Format times using Hikvision Web UI format: "HH:MM:SS+HH:MM" (plus separator)
	beginTime := formatTimeForHikvision(s.BeginTime, timezoneOffset)
	endTime := formatTimeForHikvision(s.EndTime, timezoneOffset)

	// Hikvision Web UI uses "YYYY-MM-DD+HH:MM" format for startTime/stopTime
	// where HH:MM is the TIMEZONE OFFSET (e.g., "08:00" for UTC+8), NOT the current time.
	// Example from Web UI: "startTime": "2026-07-06+08:00"
	now := time.Now()
	today := now.Format("2006-01-02") + "+" + timezoneOffset
	futureDate := now.AddDate(0, 0, 7).Format("2006-01-02") + "+" + timezoneOffset

	// Build schedule list entry
	// IMPORTANT: customAudioID must be the Hikvision device's audio ID, not the local database ID
	scheduleEntry := map[string]interface{}{
		"beginTime": beginTime,
		"endTime":   endTime,
		"playMode":  "order",
		"operation": map[string]interface{}{
			"audioSource":   "customAudio",
			"customAudioID": []int{hikvisionAudioID},
			"audioLevel":    5,
			"audioVolume":   s.Volume,
		},
	}

	// Build the plan scheme
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
