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

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// Get timezone offset from location settings
	timezoneOffset := "+08:00" // default fallback
	location, err := repositories.GetPrayerLocation()
	if err == nil && location.Timezone != "" {
		timezoneOffset = getTimezoneOffset(location.Timezone)
	}

	// Generate a stable planSchemeID
	planSchemeID := getStablePlanSchemeID(schedule)

	log.Printf("[SYNC] planSchemeID=%s, beginTime=%s, endTime=%s", planSchemeID, schedule.BeginTime, schedule.EndTime)

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
	payload := buildHikvisionSchedulePayload(schedule, timezoneOffset, planSchemeID)

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

// getTimezoneOffset converts a timezone name (e.g., "Asia/Jakarta") to offset string (e.g., "+07:00")
func getTimezoneOffset(tzName string) string {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return "+08:00"
	}
	_, offset := time.Now().In(loc).Zone()
	hours := offset / 3600
	mins := (offset % 3600) / 60
	if hours >= 0 {
		return fmt.Sprintf("+%02d:%02d", hours, mins)
	}
	return fmt.Sprintf("-%02d:%02d", -hours, mins)
}

// formatTimeForHikvision converts a time string to "HH:MM:SS+HH:MM" format.
// Handles various input formats:
//   - "HH:MM" -> "HH:MM:SS+08:00"
//   - "HH:MM:SS" -> "HH:MM:SS+08:00"
//   - "HH:MM:SS+08:00" -> "HH:MM:SS+08:00" (already formatted, timezone is replaced)
//   - "" -> returns empty string (caller should validate)
func formatTimeForHikvision(timeStr string, timezoneOffset string) string {
	if timeStr == "" {
		return ""
	}

	// Remove any existing timezone suffix (match any +/-HH:MM pattern at the end)
	re := regexp.MustCompile(`[+-]\d{2}:\d{2}$`)
	timeStr = re.ReplaceAllString(timeStr, "")

	// Count colons to determine format
	parts := strings.Split(timeStr, ":")
	switch len(parts) {
	case 2:
		// HH:MM -> HH:MM:SS
		return fmt.Sprintf("%s:%s:00%s", parts[0], parts[1], timezoneOffset)
	case 3:
		// HH:MM:SS -> HH:MM:SS
		return fmt.Sprintf("%s:%s:%s%s", parts[0], parts[1], parts[2], timezoneOffset)
	default:
		// Unknown format, return as-is with timezone
		return timeStr + timezoneOffset
	}
}

// buildHikvisionSchedulePayload builds the Hikvision ISAPI payload from our schedule model.
// Uses struct-based payload to ensure correct JSON key ordering matching the verified format.
func buildHikvisionSchedulePayload(s *models.BroadcastSchedule, timezoneOffset string, planSchemeID string) *hikvision.AddPlanSchemePayload {
	// Format times with proper timezone offset for Hikvision API
	beginTime := formatTimeForHikvision(s.BeginTime, timezoneOffset)
	endTime := formatTimeForHikvision(s.EndTime, timezoneOffset)

	// Use today's date and a near future date for startTime/stopTime in dailyScheduleInfo
	// Format: YYYY-MM-DD (date only, not time)
	// stopTime is set to 7 days from now to match the verified working payload pattern
	today := time.Now().Format("2006-01-02")
	futureDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	// Build schedule list entry
	scheduleEntry := hikvision.ScheduleEntry{
		BeginTime: beginTime,
		EndTime:   endTime,
		PlayMode:  "order",
		Operation: hikvision.Operation{
			AudioSource:   "customAudio",
			CustomAudioID: []int{s.AudioID},
			AudioLevel:    5,
			AudioVolume:   s.Volume,
		},
	}

	// Build the plan scheme
	planScheme := hikvision.BroadcastPlanScheme{
		PlanSchemeID:   planSchemeID,
		Enabled:        s.Enabled,
		PlanSchemeName: s.Name,
		AudioOutID:     []int{1},
	}

	switch s.ScheduleType {
	case "daily":
		planScheme.DailyScheduleInfo = &hikvision.DailyScheduleInfo{
			StartTime: today,
			StopTime:  futureDate,
			DailyScheduleList: []hikvision.ScheduleEntry{
				scheduleEntry,
			},
		}
	case "weekly":
		dayOfWeek := 1
		if s.DayOfWeek != nil {
			dayOfWeek = *s.DayOfWeek
		}
		planScheme.WeeklyScheduleInfo = &hikvision.WeeklyScheduleInfo{
			StartTime: today,
			StopTime:  futureDate,
			WeeklyScheduleList: []hikvision.WeeklyScheduleDay{
				{
					DayOfWeek:    dayOfWeek,
					ScheduleList: []hikvision.ScheduleEntry{scheduleEntry},
				},
			},
		}
	case "specific_date":
		dateStr := today
		if s.SpecificDate != nil && *s.SpecificDate != "" {
			dateStr = *s.SpecificDate
		}
		planScheme.DailyScheduleInfo = &hikvision.DailyScheduleInfo{
			StartTime: dateStr,
			StopTime:  dateStr,
			DailyScheduleList: []hikvision.ScheduleEntry{
				scheduleEntry,
			},
		}
	}

	payload := &hikvision.AddPlanSchemePayload{
		BroadcastPlanSchemeList: []hikvision.BroadcastPlanScheme{
			planScheme,
		},
		TerminalInfoList: []hikvision.TerminalInfo{
			{
				TerminalID: 1,
				AudioOutID: []int{1},
			},
		},
	}

	return payload
}
