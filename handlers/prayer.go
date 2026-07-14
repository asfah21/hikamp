package handlers

import (
	"database/sql"
	"ego/models"
	"ego/services"
	"log"
	"net/http"
	"strconv"
)

// AdminPrayerSetting renders the prayer location settings page
func AdminPrayerSetting(w http.ResponseWriter, r *http.Request) {
	location, err := services.GetPrayerLocation()
	if err != nil {
		location = nil
	}

	data := map[string]interface{}{
		"Location": location,
	}
	RenderDashboard(w, r, "prayer_setting", data)
}

// AdminPrayerTime renders the prayer times page
func AdminPrayerTime(w http.ResponseWriter, r *http.Request) {
	location, err := services.GetPrayerLocation()
	if err != nil {
		location = nil
	}

	var prayerTimes []models.PrayerTime
	if location != nil {
		// Get all available prayer times (up to 365 days)
		prayerTimes, _ = services.GetUpcomingPrayerTimes(365)
	}

	data := map[string]interface{}{
		"Location":    location,
		"PrayerTimes": prayerTimes,
	}
	RenderDashboard(w, r, "prayer_time", data)
}

// AdminPrayerBroadcast renders the prayer broadcast settings page
func AdminPrayerBroadcast(w http.ResponseWriter, r *http.Request) {
	location, err := services.GetPrayerLocation()
	if err != nil {
		location = nil
	}

	broadcastConfigs, _ := services.GetPrayerBroadcastConfigs()
	devices, _ := services.GetAllDevices()
	audioFiles, _ := services.GetAllAudioFiles()

	// Ensure we always have 5 configs (one per prayer) to avoid template index out of range errors
	prayerOrder := []string{"fajr", "dhuhr", "asr", "maghrib", "isha"}
	configMap := make(map[string]*models.PrayerBroadcastConfig)
	for i := range broadcastConfigs {
		configMap[broadcastConfigs[i].Prayer] = &broadcastConfigs[i]
	}

	// Build a slice of exactly 5 configs in the correct order, filling defaults for missing ones
	fullConfigs := make([]models.PrayerBroadcastConfig, 0, 5)
	for _, prayer := range prayerOrder {
		if cfg, ok := configMap[prayer]; ok {
			fullConfigs = append(fullConfigs, *cfg)
		} else {
			fullConfigs = append(fullConfigs, models.PrayerBroadcastConfig{
				Prayer:   prayer,
				AudioID:  sql.NullInt64{Valid: false},
				DeviceID: sql.NullInt64{Valid: false},
				Volume:   50,
				Duration: 5,
				Enabled:  false,
			})
		}
	}

	data := map[string]interface{}{
		"Location":         location,
		"BroadcastConfigs": fullConfigs,
		"Devices":          devices,
		"Audio":            audioFiles,
	}
	RenderDashboard(w, r, "prayer_broadcast", data)
}

// detectTimezoneFromLongitude determines the IANA timezone name from longitude.
// Indonesia: 95-141°E covers WIB (UTC+7), WITA (UTC+8), WIT (UTC+9).
// For other regions, maps to common timezones by offset.
func detectTimezoneFromLongitude(longitude float64) string {
	// Indonesia timezones
	if longitude >= 95 && longitude < 120 {
		return "Asia/Jakarta" // WIB (UTC+7)
	}
	if longitude >= 120 && longitude < 138 {
		return "Asia/Makassar" // WITA (UTC+8)
	}
	if longitude >= 138 && longitude <= 141 {
		return "Asia/Jayapura" // WIT (UTC+9)
	}

	// For other regions, calculate offset and map to common timezones
	// Round to nearest half hour
	offset := (longitude + 7.5) / 15
	if offset > 12 {
		offset = 12
	}
	if offset < -12 {
		offset = -12
	}

	// Map common offsets to IANA timezones
	// India is UTC+5:30, Nepal is UTC+5:45 — handle separately
	if offset > 5.25 && offset < 5.75 {
		return "Asia/Kolkata"
	}
	if offset > 5.6 && offset < 5.9 {
		return "Asia/Kathmandu"
	}

	// Round to nearest hour for other timezones
	rounded := int(offset + 0.5)
	if rounded > 12 {
		rounded = 12
	}
	if rounded < -12 {
		rounded = -12
	}

	switch rounded {
	case 7:
		return "Asia/Jakarta"
	case 8:
		return "Asia/Makassar"
	case 9:
		return "Asia/Jayapura"
	case 3:
		return "Asia/Riyadh"
	case 4:
		return "Asia/Dubai"
	case 5:
		return "Asia/Karachi"
	case 6:
		return "Asia/Dhaka"
	case 10:
		return "Asia/Seoul"
	case 11:
		return "Pacific/Guadalcanal"
	case 12:
		return "Pacific/Auckland"
	case -5:
		return "America/New_York"
	case -6:
		return "America/Chicago"
	case -7:
		return "America/Denver"
	case -8:
		return "America/Los_Angeles"
	case 0:
		return "Europe/London"
	case 1:
		return "Europe/Paris"
	case 2:
		return "Europe/Helsinki"
	default:
		return "Asia/Makassar"
	}
}

// AdminPrayerSave handles prayer location save and auto-generates prayer times
func AdminPrayerSave(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		latitude, _ := strconv.ParseFloat(r.FormValue("latitude"), 64)
		longitude, _ := strconv.ParseFloat(r.FormValue("longitude"), 64)

		// Auto-detect timezone from longitude
		// Each 15° = 1 hour offset, rounded to nearest half hour
		timezone := detectTimezoneFromLongitude(longitude)

		location := &models.PrayerLocation{
			Latitude:  latitude,
			Longitude: longitude,
			Timezone:  timezone,
			Method:    0, // no longer used
		}

		err := services.SavePrayerLocation(location)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get the saved location with ID
		savedLocation, err := services.GetPrayerLocation()
		if err == nil {
			// Auto-generate prayer times for the next 30 days
			go services.AutoGeneratePrayerTimes(savedLocation, 30)
		}

		// Redirect back to prayer setting page so content is refreshed
		w.Header().Set("HX-Redirect", "/admin/prayer/setting")
		w.WriteHeader(http.StatusOK)
		return
	}
}

// AdminPrayerGenerate handles manual prayer times generation
func AdminPrayerGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		location, err := services.GetPrayerLocation()
		if err != nil {
			setHXTriggerToast(w, "Please set a location first")
			w.WriteHeader(http.StatusOK)
			return
		}

		daysStr := r.FormValue("days")
		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
				days = d
			}
		}

		err = services.AutoGeneratePrayerTimes(location, days)
		if err != nil {
			setHXTriggerToast(w, "Failed to generate prayer times: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		setHXTriggerToast(w, "Prayer times generated for "+strconv.Itoa(days)+" days", true)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// AdminPrayerBroadcastSave handles saving prayer broadcast configs
func AdminPrayerBroadcastSave(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		prayers := []string{"fajr", "dhuhr", "asr", "maghrib", "isha"}
		for _, prayer := range prayers {
			audioID, _ := strconv.Atoi(r.FormValue("audio_" + prayer))
			deviceID, _ := strconv.Atoi(r.FormValue("device_" + prayer))
			volume, _ := strconv.Atoi(r.FormValue("volume_" + prayer))
			enabled := r.FormValue("enabled_"+prayer) == "on"

			cfg := &models.PrayerBroadcastConfig{
				Prayer:   prayer,
				AudioID:  sql.NullInt64{Int64: int64(audioID), Valid: audioID > 0},
				DeviceID: sql.NullInt64{Int64: int64(deviceID), Valid: deviceID > 0},
				Volume:   volume,
				Duration: 0,
				Enabled:  enabled,
			}
			if err := services.SavePrayerBroadcastConfig(cfg); err != nil {
				log.Printf("[PRAYER BROADCAST] Failed to save config for %s: %v", prayer, err)
			}
		}

		// After saving broadcast configs, create weekly schedules on devices
		location, err := services.GetPrayerLocation()
		var messages []string
		if err == nil && location != nil {
			messages = services.CreatePrayerSchedules(location, 30)
		} else {
			messages = []string{"Location not set. Please set location in Prayer Settings first."}
		}

		// Build toast message from all messages
		toastMsg := "Settings saved."
		for _, m := range messages {
			toastMsg += " " + m
		}
		setHXTriggerToast(w, toastMsg, true)
		w.WriteHeader(http.StatusOK)
		return

	}
}

// AdminPrayerCreateSchedules creates weekly prayer schedules on devices
func AdminPrayerCreateSchedules(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		location, err := services.GetPrayerLocation()
		if err != nil {
			setHXTriggerToast(w, "Please set a location first")
			w.WriteHeader(http.StatusOK)
			return
		}

		daysStr := r.FormValue("days")
		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
				days = d
			}
		}

		messages := services.CreatePrayerSchedules(location, days)
		toastMsg := ""
		for _, m := range messages {
			toastMsg += " " + m
		}
		setHXTriggerToast(w, toastMsg, true)
		w.WriteHeader(http.StatusOK)
		return
	}
}
