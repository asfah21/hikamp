package services

import (
	"ego/models"
	"ego/repositories"
	"fmt"
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
	results := GeneratePrayerTimesForRange(location.Latitude, location.Longitude, location.Timezone, location.Method, startDate, endDate)

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

// CreatePrayerSchedules creates broadcast schedules from prayer times for enabled configs
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
		return err
	}

	for _, pt := range prayerTimes {
		for _, cfg := range configs {
			if !cfg.Enabled || cfg.AudioID == 0 || cfg.DeviceID == 0 {
				continue
			}

			// Get the prayer time value based on config prayer name
			var prayerTime string
			switch cfg.Prayer {
			case "fajr":
				prayerTime = pt.Fajr
			case "dhuhr":
				prayerTime = pt.Dhuhr
			case "asr":
				prayerTime = pt.Asr
			case "maghrib":
				prayerTime = pt.Maghrib
			case "isha":
				prayerTime = pt.Isha
			}

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

			// Create a daily schedule for this prayer time
			schedule := &models.BroadcastSchedule{
				Name:         "Prayer: " + models.PrayerNames[cfg.Prayer] + " - " + pt.Date,
				AudioID:      cfg.AudioID,
				DeviceID:     cfg.DeviceID,
				ScheduleType: "daily",
				BeginTime:    prayerTime,
				EndTime:      endTime,
				Volume:       cfg.Volume,
				Enabled:      true,
			}

			_, err := repositories.CreateSchedule(schedule)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
