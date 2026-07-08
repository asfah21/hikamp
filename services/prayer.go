package services

import (
	"ego/internal/hikvision"
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

// CreatePrayerSchedules creates 5 weekly broadcast schedules on Hikvision devices
// (one per prayer time: Fajr, Dhuhr, Asr, Maghrib, Isha).
// Each schedule repeats every day of the week (dayOfWeek 1-7).
// Instead of creating 150+ individual daily schedules in the database,
// this directly syncs 5 weekly schedules to each device.
func CreatePrayerSchedules(location *models.PrayerLocation, days int) error {
	configs, err := repositories.GetPrayerBroadcastConfigs()
	if err != nil {
		return err
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
		return nil // not a fatal error — configs are still saved
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
		return nil
	}

	// Get timezone offset
	timezoneOffset := "08:00"
	if location.Timezone != "" {
		timezoneOffset = getTimezoneOffset(location.Timezone)
	}

	// For each enabled config, create a weekly schedule on the device
	for _, cfg := range configs {
		if !cfg.Enabled || cfg.AudioID == 0 || cfg.DeviceID == 0 {
			continue
		}

		prayerTime := todayTimes[cfg.Prayer]
		if prayerTime == "" {
			continue
		}

		// Calculate end time: prayer time + broadcast duration (default 5 minutes)
		endTime := prayerTime
		if parts := strings.Split(prayerTime, ":"); len(parts) >= 2 {
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			totalMin := h*60 + m + 5 // add 5 minutes
			endH := totalMin / 60
			endM := totalMin % 60
			if endH >= 24 {
				endH = 23
				endM = 59
			}
			if len(parts) == 2 {
				endTime = fmt.Sprintf("%02d:%02d", endH, endM)
			} else {
				s, _ := strconv.Atoi(parts[2])
				endTime = fmt.Sprintf("%02d:%02d:%02d", endH, endM, s)
			}
		}

		// Get device info
		device, err := repositories.GetDeviceByID(cfg.DeviceID)
		if err != nil {
			log.Printf("[PRAYER SCHEDULE] Device ID %d not found: %v", cfg.DeviceID, err)
			continue
		}

		// Get audio file to get HikvisionAudioID
		audioFile, err := repositories.GetAudioFileByID(cfg.AudioID)
		if err != nil {
			log.Printf("[PRAYER SCHEDULE] Audio ID %d not found: %v", cfg.AudioID, err)
			continue
		}
		if audioFile.HikvisionAudioID == nil || *audioFile.HikvisionAudioID == 0 {
			log.Printf("[PRAYER SCHEDULE] Audio '%s' has no Hikvision audio ID. Upload to device first.", audioFile.Name)
			continue
		}

		client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

		// Build a weekly schedule that repeats every day (dayOfWeek 1-7)
		prayerName := models.PrayerNames[cfg.Prayer]
		planSchemeID := fmt.Sprintf("prayer_%s_%s", cfg.Prayer, device.IPAddress)

		// Format times for Hikvision
		beginTimeHik := formatTimeForHikvision(prayerTime, timezoneOffset)
		endTimeHik := formatTimeForHikvision(endTime, timezoneOffset)

		// Date range: today + 7 days (weekly repeat)
		today := now.Format("2006-01-02") + "+" + timezoneOffset
		futureDate := now.AddDate(0, 0, 7).Format("2006-01-02") + "+" + timezoneOffset

		// Build weekly schedule with all 7 days
		weeklyScheduleList := make([]map[string]interface{}, 0, 7)
		for day := 1; day <= 7; day++ {
			weeklyScheduleList = append(weeklyScheduleList, map[string]interface{}{
				"dayOfWeek": day,
				"scheduleList": []map[string]interface{}{
					{
						"beginTime": beginTimeHik,
						"endTime":   endTimeHik,
						"playMode":  "order",
						"operation": map[string]interface{}{
							"audioSource":   "customAudio",
							"customAudioID": []int{*audioFile.HikvisionAudioID},
							"audioLevel":    5,
							"audioVolume":   cfg.Volume,
						},
					},
				},
			})
		}

		payload := map[string]interface{}{
			"broadcastPlanSchemeList": []map[string]interface{}{
				{
					"planSchemeID":   planSchemeID,
					"planSchemeName": "Prayer: " + prayerName,
					"enabled":        true,
					"audioOutID":     []int{1},
					"weeklyScheduleInfo": map[string]interface{}{
						"startTime":          today,
						"stopTime":           futureDate,
						"weeklyScheduleList": weeklyScheduleList,
					},
				},
			},
			"terminalInfoList": []map[string]interface{}{
				{
					"terminalID": 1,
					"audioOutID": []int{1},
				},
			},
		}

		// First try to delete existing scheme with same ID
		_ = client.DeletePlanScheme(planSchemeID)
		time.Sleep(300 * time.Millisecond)

		// Create the new schedule
		err = client.CreateSchedule(payload)
		if err != nil {
			log.Printf("[PRAYER SCHEDULE] Failed to create schedule for %s on device %s: %v", prayerName, device.Name, err)
			continue
		}

		log.Printf("[PRAYER SCHEDULE] Created weekly schedule for %s on device %s (planSchemeID: %s)", prayerName, device.Name, planSchemeID)
	}

	return nil
}
