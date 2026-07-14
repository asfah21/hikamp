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

// CreatePrayerSchedules creates daily broadcast schedules in the database
// (one per prayer time: Fajr, Dhuhr, Asr, Maghrib, Isha).
// Each schedule uses schedule_type "daily" so it repeats every day.
// Saves to the broadcast_schedules table (admin/schedules) instead of directly
// sending to the Hikvision device. The user can then review and sync manually.
// Returns a list of human-readable warnings/messages for the UI.
func CreatePrayerSchedules(location *models.PrayerLocation, days int) []string {
	configs, err := repositories.GetPrayerBroadcastConfigs()
	if err != nil {
		return []string{fmt.Sprintf("Failed to load broadcast configs: %v", err)}
	}

	now := time.Now()
	loc, err := time.LoadLocation(location.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now = now.In(loc)

	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, days-1).Format("2006-01-02")

	prayerTimes, err := repositories.GetPrayerTimes(startDate, endDate)
	if err != nil {
		log.Printf("[PRAYER SCHEDULE] No prayer times found for range %s to %s: %v", startDate, endDate, err)
		return []string{"No prayer times found. Please generate prayer times first (Prayer Times > Generate)."}
	}

	// Group prayer times by prayer name to get the first occurrence
	// (all days have the same times, just use today's)
	todayTimes := make(map[string]string)
	for _, pt := range prayerTimes {
		if pt.Date == startDate {
			todayTimes["fajr"] = pt.Fajr
			todayTimes["dhuhr"] = pt.Dhuhr
			todayTimes["asr"] = pt.Asr
			todayTimes["maghrib"] = pt.Maghrib
			todayTimes["isha"] = pt.Isha
			break
		}
	}

	// If today's times not found, use the first available day
	if len(todayTimes) == 0 && len(prayerTimes) > 0 {
		pt := prayerTimes[0]
		todayTimes["fajr"] = pt.Fajr
		todayTimes["dhuhr"] = pt.Dhuhr
		todayTimes["asr"] = pt.Asr
		todayTimes["maghrib"] = pt.Maghrib
		todayTimes["isha"] = pt.Isha
	}

	// If still no prayer times, just log and return (configs are saved)
	if len(todayTimes) == 0 {
		log.Printf("[PRAYER SCHEDULE] No prayer times available. Generate prayer times first.")
		return []string{"No prayer times available. Please generate prayer times first (Prayer Times > Generate)."}
	}

	var messages []string
	successCount := 0
	skipCount := 0

	// For each enabled config, create a daily schedule in the database
	for _, cfg := range configs {
		prayerName := models.PrayerNames[cfg.Prayer]

		if !cfg.Enabled {
			skipCount++
			continue
		}
		if !cfg.AudioID.Valid {
			messages = append(messages, fmt.Sprintf("%s: No audio selected.", prayerName))
			continue
		}
		if !cfg.DeviceID.Valid {
			messages = append(messages, fmt.Sprintf("%s: No device selected.", prayerName))
			continue
		}

		prayerTime := todayTimes[cfg.Prayer]
		if prayerTime == "" {
			messages = append(messages, fmt.Sprintf("%s: Prayer time not found.", prayerName))
			continue
		}

		// Ensure prayer time is in HH:MM:SS format (pad :00 if only HH:MM)
		if strings.Count(prayerTime, ":") == 1 {
			prayerTime = prayerTime + ":00"
		}

		// Lookup audio file to get its duration in seconds
		audioID := int(cfg.AudioID.Int64)
		audioFile, err := repositories.GetAudioFileByID(audioID)
		audioDurationSec := 0
		if err == nil && audioFile != nil {
			audioDurationSec = audioFile.Duration
		}
		if audioDurationSec <= 0 {
			audioDurationSec = 300 // default 5 minutes = 300 seconds
		}

		// Calculate end time: prayer time + audio duration (in seconds)
		endTime := prayerTime
		if parts := strings.Split(prayerTime, ":"); len(parts) >= 2 {
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			s := 0
			if len(parts) >= 3 {
				s, _ = strconv.Atoi(parts[2])
			}
			totalSec := h*3600 + m*60 + s + audioDurationSec
			endH := (totalSec / 3600) % 24
			endM := (totalSec % 3600) / 60
			endS := totalSec % 60
			endTime = fmt.Sprintf("%02d:%02d:%02d", endH, endM, endS)
		}

		deviceID := int(cfg.DeviceID.Int64)

		// Create schedule in database (daily type = repeats every day)
		schedule := &models.BroadcastSchedule{
			Name:         "Prayer: " + prayerName,
			ScheduleType: "daily",
			Enabled:      true,
			Entries: []models.ScheduleEntry{
				{
					AudioID:   audioID,
					BeginTime: prayerTime,
					EndTime:   endTime,
					Volume:    cfg.Volume,
				},
			},
			Devices: []models.ScheduleDevice{
				{
					DeviceID: deviceID,
				},
			},
		}

		_, err = repositories.CreateSchedule(schedule)
		if err != nil {
			messages = append(messages, fmt.Sprintf("%s: Failed to save schedule: %v", prayerName, err))
			continue
		}

		successCount++
		log.Printf("[PRAYER SCHEDULE] Created daily schedule for %s (device ID: %d, audio ID: %d)", prayerName, deviceID, audioID)
	}

	if successCount > 0 {
		messages = append(messages, fmt.Sprintf("%d prayer schedule(s) saved to database. Go to Schedules menu to review and sync.", successCount))
	}
	if skipCount > 0 {
		messages = append(messages, fmt.Sprintf("%d prayer(s) skipped (not enabled).", skipCount))
	}

	return messages
}
