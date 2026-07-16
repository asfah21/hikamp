package services

import (
	"ego/models"
	"ego/repositories"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// GetPrayerLocation returns the prayer location
func GetPrayerLocation() (*models.PrayerLocation, error) {
	return repositories.GetPrayerLocation()
}

// SavePrayerLocation saves the prayer location
func SavePrayerLocation(p *models.PrayerLocation) error {
	return repositories.SavePrayerLocation(p)
}

// GetPrayerTimes returns prayer times for a date range
func GetPrayerTimes(startDate, endDate string) ([]models.PrayerTime, error) {
	return repositories.GetPrayerTimes(startDate, endDate)
}

// GenerateAndSavePrayerTimes generates prayer times and saves them to database
func GenerateAndSavePrayerTimes(location *models.PrayerLocation, startDate, endDate time.Time) error {
	results := GeneratePrayerTimesForRange(location.Latitude, location.Longitude, location.Timezone, startDate, endDate)

	for _, r := range results {
		pt := &models.PrayerTime{
			Date:       r.Date,
			Fajr:       r.Fajr,
			Dhuhr:      r.Dhuhr,
			Asr:        r.Asr,
			Maghrib:    r.Maghrib,
			Isha:       r.Isha,
			LocationID: location.ID,
		}
		err := repositories.SavePrayerTime(pt)
		if err != nil {
			return err
		}
	}
	return nil
}

// AutoGeneratePrayerTimes generates prayer times for the next N days
func AutoGeneratePrayerTimes(location *models.PrayerLocation, days int) error {
	now := time.Now()
	loc, err := time.LoadLocation(location.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now = now.In(loc)

	startDate := now
	endDate := now.AddDate(0, 0, days-1)

	return GenerateAndSavePrayerTimes(location, startDate, endDate)
}

// GetUpcomingPrayerTimes returns prayer times for today and next few days
func GetUpcomingPrayerTimes(days int) ([]models.PrayerTime, error) {
	// Use location timezone if available
	location, err := repositories.GetPrayerLocation()
	now := time.Now()
	if err == nil && location != nil {
		loc, tzErr := time.LoadLocation(location.Timezone)
		if tzErr == nil {
			now = now.In(loc)
		}
	}
	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, days-1).Format("2006-01-02")
	return repositories.GetPrayerTimes(startDate, endDate)
}

// GetPrayerBroadcastConfigs returns all prayer broadcast configs
func GetPrayerBroadcastConfigs() ([]models.PrayerBroadcastConfig, error) {
	return repositories.GetPrayerBroadcastConfigs()
}

// SavePrayerBroadcastConfig saves a prayer broadcast config
func SavePrayerBroadcastConfig(c *models.PrayerBroadcastConfig) error {
	return repositories.SavePrayerBroadcastConfig(c)
}

// CreatePrayerSchedules creates a single weekly broadcast schedule with all prayer entries
// (Fajr, Dhuhr, Asr, Maghrib, Isha) for today + the next 6 days (7 days total).
// The schedule uses schedule_type "weekly" with up to 35 entries (5 prayers × 7 days).
// Before creating, it deletes any existing schedule with source = "prayer" to avoid duplicates.
// Saves to the broadcast_schedules table (admin/schedules) for review and manual sync.
// Returns a list of human-readable warnings/messages for the UI.
func CreatePrayerSchedules(location *models.PrayerLocation, days int) []string {
	configs, err := repositories.GetPrayerBroadcastConfigs()
	if err != nil {
		return []string{fmt.Sprintf("Failed to load broadcast configs: %v", err)}
	}

	loc, err := time.LoadLocation(location.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	startDate := now.Format("2006-01-02")

	// Load prayer times for the full range (today + next 6 days = 7 days for weekly)
	prayerDaysCount := 7
	if days < prayerDaysCount {
		prayerDaysCount = days
	}
	prayerEndDate := now.AddDate(0, 0, prayerDaysCount-1).Format("2006-01-02")

	prayerTimes, err := repositories.GetPrayerTimes(startDate, prayerEndDate)
	if err != nil {
		log.Printf("[PRAYER SCHEDULE] No prayer times found for range %s to %s: %v", startDate, prayerEndDate, err)
		return []string{"No prayer times found. Please generate prayer times first (Prayer Times > Generate)."}
	}

	// Group prayer times by date for easy lookup
	prayerTimesByDate := make(map[string]map[string]string)
	for _, pt := range prayerTimes {
		prayerTimesByDate[pt.Date] = map[string]string{
			"fajr":    pt.Fajr,
			"dhuhr":   pt.Dhuhr,
			"asr":     pt.Asr,
			"maghrib": pt.Maghrib,
			"isha":    pt.Isha,
		}
	}

	// Collect all enabled configs with valid audio/device
	type validConfig struct {
		config           models.PrayerBroadcastConfig
		prayerName       string
		audioDurationSec int
	}
	var validConfigs []validConfig
	hasEnabled := false

	for _, cfg := range configs {
		prayerName := models.PrayerNames[cfg.Prayer]

		if !cfg.Enabled {
			continue
		}
		hasEnabled = true
		if !cfg.AudioID.Valid {
			continue
		}
		if !cfg.DeviceID.Valid {
			continue
		}

		// Lookup audio file to get its duration in seconds
		audioID := int(cfg.AudioID.Int64)
		audioFile, err := repositories.GetAudioFileByID(audioID)
		audioDurationSec := 300 // default 5 minutes
		if err == nil && audioFile != nil && audioFile.Duration > 0 {
			audioDurationSec = audioFile.Duration
		}

		validConfigs = append(validConfigs, validConfig{
			config:           cfg,
			prayerName:       prayerName,
			audioDurationSec: audioDurationSec,
		})
	}

	if !hasEnabled {
		return []string{"No prayer broadcast settings are enabled. Enable at least one prayer first."}
	}
	if len(validConfigs) == 0 {
		return []string{"No valid broadcast configs found. Make sure each enabled prayer has an audio file and device selected."}
	}

	// Detect which devices are used across all configs
	deviceIDs := make(map[int]bool)
	for _, vc := range validConfigs {
		deviceIDs[int(vc.config.DeviceID.Int64)] = true
	}

	var messages []string
	successCount := 0

	// For each device, delete old prayer schedule and create a new one
	for deviceID := range deviceIDs {
		// Delete existing prayer-generated schedule for this device
		if err := repositories.DeleteSchedulesBySource("prayer"); err != nil {
			log.Printf("[PRAYER SCHEDULE] Failed to delete old prayer schedules: %v", err)
		}

		// Build all entries grouped by day of week (1=Monday ... 7=Sunday)
		type dayEntry struct {
			dayOfWeek int
			beginTime string
			endTime   string
			audioID   int
			volume    int
		}

		var allEntries []dayEntry

		// For each day in our prayer times range, add entries for all valid configs
		for dayOffset := 0; dayOffset < prayerDaysCount; dayOffset++ {
			date := now.AddDate(0, 0, dayOffset).Format("2006-01-02")
			dayTimes, ok := prayerTimesByDate[date]
			if !ok {
				continue
			}

			// Calculate day of week (1=Monday ... 7=Sunday)
			dayOfWeek := int(now.AddDate(0, 0, dayOffset).Weekday())
			if dayOfWeek == 0 {
				dayOfWeek = 7 // Go's Sunday=0, we need Sunday=7
			}

			for _, vc := range validConfigs {
				if int(vc.config.DeviceID.Int64) != deviceID {
					continue
				}

				prayerTime := dayTimes[vc.config.Prayer]
				if prayerTime == "" {
					continue
				}

				// Ensure prayer time is in HH:MM:SS format
				if strings.Count(prayerTime, ":") == 1 {
					prayerTime = prayerTime + ":00"
				}

				// Calculate end time: prayer time + audio duration
				endTime := prayerTime
				if parts := strings.Split(prayerTime, ":"); len(parts) >= 2 {
					h, _ := strconv.Atoi(parts[0])
					m, _ := strconv.Atoi(parts[1])
					s := 0
					if len(parts) >= 3 {
						s, _ = strconv.Atoi(parts[2])
					}
					totalSec := h*3600 + m*60 + s + vc.audioDurationSec
					endH := (totalSec / 3600) % 24
					endM := (totalSec % 3600) / 60
					endS := totalSec % 60
					endTime = fmt.Sprintf("%02d:%02d:%02d", endH, endM, endS)
				}

				allEntries = append(allEntries, dayEntry{
					dayOfWeek: dayOfWeek,
					beginTime: prayerTime,
					endTime:   endTime,
					audioID:   int(vc.config.AudioID.Int64),
					volume:    vc.config.Volume,
				})
			}
		}

		if len(allEntries) == 0 {
			messages = append(messages, fmt.Sprintf("Device %d: No prayer times available for the selected days.", deviceID))
			continue
		}

		// Group entries by dayOfWeek for the weekly schedule format
		weeklyScheduleMap := make(map[int][]map[string]interface{})
		for _, e := range allEntries {
			entryMap := map[string]interface{}{
				"beginTime": e.beginTime,
				"endTime":   e.endTime,
				"playMode":  "order",
				"operation": map[string]interface{}{
					"audioSource":   "customAudio",
					"customAudioID": []int{e.audioID},
					"audioLevel":    5,
					"audioVolume":   e.volume,
				},
			}
			weeklyScheduleMap[e.dayOfWeek] = append(weeklyScheduleMap[e.dayOfWeek], entryMap)
		}

		// Build weeklyScheduleList in day order
		type weekEntry struct {
			dayOfWeek    int
			scheduleList []map[string]interface{}
		}
		var weeklyList []weekEntry
		for d := 1; d <= 7; d++ {
			if list, ok := weeklyScheduleMap[d]; ok {
				weeklyList = append(weeklyList, weekEntry{dayOfWeek: d, scheduleList: list})
			}
		}

		// Build schedule entries (for local DB storage) with day_of_week per entry
		var dbEntries []models.ScheduleEntry
		for _, e := range allEntries {
			dow := e.dayOfWeek
			dbEntries = append(dbEntries, models.ScheduleEntry{
				AudioID:   e.audioID,
				BeginTime: e.beginTime,
				EndTime:   e.endTime,
				Volume:    e.volume,
				DayOfWeek: &dow,
			})
		}

		scheduleStart := now.Format("2006-01-02")
		scheduleEnd := now.AddDate(0, 0, prayerDaysCount-1).Format("2006-01-02")

		// Create single weekly schedule with all prayer entries
		schedule := &models.BroadcastSchedule{
			Name:         "Prayer Broadcasts",
			ScheduleType: "weekly",
			Enabled:      true,
			Source:       "prayer",
			StartDate:    &scheduleStart,
			EndDate:      &scheduleEnd,
			Entries:      dbEntries,
			Devices: []models.ScheduleDevice{
				{
					DeviceID: deviceID,
				},
			},
		}

		_, err = repositories.CreateSchedule(schedule)
		if err != nil {
			messages = append(messages, fmt.Sprintf("Device %d: Failed to save schedule: %v", deviceID, err))
			continue
		}

		successCount++
		log.Printf("[PRAYER SCHEDULE] Created weekly schedule for device ID %d with %d entries (%d days)", deviceID, len(allEntries), prayerDaysCount)
	}

	if successCount > 0 {
		messages = append(messages, fmt.Sprintf("Prayer schedule updated with %d entries across %d device(s). Go to Schedules menu to review and sync.", len(validConfigs)*prayerDaysCount, len(deviceIDs)))
	}

	return messages
}
