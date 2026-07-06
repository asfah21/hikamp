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
// Uses a stable planSchemeID and deletes any existing schedule with the same ID first
// to prevent duplicates on re-sync.
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
	if audioFile.HikvisionAudioID == 0 {
		return fmt.Errorf("audio file '%s' has no Hikvision audio ID. Upload the audio to the device first", audioFile.Name)
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// Get timezone offset from location settings
	// Note: getTimezoneOffset returns WITHOUT "+" prefix (Web UI format: "08:00")
	timezoneOffset := "08:00" // default fallback
	location, err := repositories.GetPrayerLocation()
	if err == nil && location.Timezone != "" {
		timezoneOffset = getTimezoneOffset(location.Timezone)
	}

	// Generate a stable planSchemeID
	planSchemeID := getStablePlanSchemeID(schedule)

	log.Printf("[SYNC] planSchemeID=%s, beginTime=%s, endTime=%s, hikvisionAudioID=%d", planSchemeID, schedule.BeginTime, schedule.EndTime, audioFile.HikvisionAudioID)

	// Step 1: Delete existing plan scheme with the same ID (if any)
	// This prevents "duplicate" or conflict errors on re-sync.
	// Ignore error if delete fails (e.g., scheme doesn't exist yet)
	err = client.DeletePlanScheme(planSchemeID)
	if err != nil {
		log.Printf("[SYNC] DeletePlanScheme (expected if first sync): %v", err)
	} else {
		log.Printf("[SYNC] Successfully deleted existing plan scheme: %s", planSchemeID)
		// Wait a moment for the device to process the deletion
		time.Sleep(500 * time.Millisecond)
	}

	// Step 2: Build and send the new schedule payload
	payload := buildHikvisionSchedulePayload(schedule, timezoneOffset, planSchemeID, audioFile.HikvisionAudioID)

	err = client.CreateSchedule(payload)
	if err != nil {
		return fmt.Errorf("create schedule failed: %w", err)
	}

	// Step 3: Verify the schedule was created by searching for it
	log.Printf("[SYNC] Schedule created successfully, verifying...")
	searchResult, err := client.SearchPlanScheme()
	if err != nil {
		log.Printf("[SYNC] Warning: verification search failed: %v", err)
	} else {
		log.Printf("[SYNC] Verification search result: %+v", searchResult)
	}

	return nil
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
// Key differences from standard camelCase:
//   - "dailyscheduleInfo" (lowercase 's') — matches Web UI, NOT "dailyScheduleInfo"
//   - startTime/stopTime format: "YYYY-MM-DD+HH:MM" (with "+" separator)
//   - beginTime/endTime format: "HH:MM:SS+HH:MM" (plus separator)
//
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
	// Note: Web UI does NOT send "planSchemeName" field
	planScheme := map[string]interface{}{
		"planSchemeID": planSchemeID,
		"enabled":      s.Enabled,
		"audioOutID":   []int{1},
	}

	switch s.ScheduleType {
	case "daily":
		// IMPORTANT: Web UI uses "dailyscheduleInfo" (lowercase 's'), NOT "dailyScheduleInfo"
		planScheme["dailyscheduleInfo"] = map[string]interface{}{
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
		// IMPORTANT: Web UI uses "dailyscheduleInfo" (lowercase 's'), NOT "dailyScheduleInfo"
		planScheme["dailyscheduleInfo"] = map[string]interface{}{
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
