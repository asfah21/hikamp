package handlers

import (
	"database/sql"
	"ego/models"
	"ego/services"
	"ego/templates"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// AdminDashboard renders the dashboard page
func AdminDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := services.GetDashboardData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	RenderDashboard(w, r, "dashboard", data)
}

// AdminDevices renders the device management page
func AdminDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := services.GetAllDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RenderDashboard(w, r, "devices", devices)
}

// AdminDevicesCreate handles device creation
func AdminDevicesCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		port, _ := strconv.Atoi(r.FormValue("port"))
		enabled := r.FormValue("enabled") == "on"

		device := &models.Device{
			Name:      r.FormValue("name"),
			IPAddress: r.FormValue("ip_address"),
			Port:      port,
			Username:  r.FormValue("username"),
			Password:  r.FormValue("password"),
			Location:  r.FormValue("location"),
			Enabled:   enabled,
		}

		_, err := services.CreateDevice(device)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("HX-Trigger", `{"toast":"Device created successfully","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	// If HTMX request, render only the modal content without layout
	if r.Header.Get("HX-Request") == "true" {
		user, _ := r.Context().Value("user").(*models.User)
		if user == nil {
			user = &models.User{Name: "Admin", Username: "admin"}
		}
		pageData := templates.PageData{
			Title: "Add Device",
			User:  user,
			Data:  nil,
		}
		templates.RenderPartial(w, "dashboard", "devices_form", pageData)
		return
	}

	RenderDashboard(w, r, "devices_form", nil)
}

// AdminDevicesEdit handles device edit form
func AdminDevicesEdit(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/devices/edit/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method == "POST" {
		port, _ := strconv.Atoi(r.FormValue("port"))
		enabled := r.FormValue("enabled") == "on"

		device := &models.Device{
			ID:        id,
			Name:      r.FormValue("name"),
			IPAddress: r.FormValue("ip_address"),
			Port:      port,
			Username:  r.FormValue("username"),
			Password:  r.FormValue("password"),
			Location:  r.FormValue("location"),
			Enabled:   enabled,
		}

		err := services.UpdateDevice(device)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("HX-Trigger", `{"toast":"Device updated successfully","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	device, err := services.GetDeviceByID(id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// If HTMX request, render only the modal content without layout
	if r.Header.Get("HX-Request") == "true" {
		user, _ := r.Context().Value("user").(*models.User)
		if user == nil {
			user = &models.User{Name: "Admin", Username: "admin"}
		}
		pageData := templates.PageData{
			Title: "Edit Device",
			User:  user,
			Data:  device,
		}
		templates.RenderPartial(w, "dashboard", "devices_form", pageData)
		return
	}

	RenderDashboard(w, r, "devices_form", device)
}

// AdminDevicesDelete handles device deletion
func AdminDevicesDelete(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/devices/delete/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = services.DeleteDevice(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":"Device deleted successfully","reload":"true"}`)
	w.WriteHeader(http.StatusOK)
}

// AdminDevicesTestConnection tests connection to a device
func AdminDevicesTestConnection(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/devices/test/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	device, err := services.GetDeviceByID(id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	info, err := services.TestDeviceConnection(device.IPAddress, device.Port, device.Username, device.Password)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"toast":"Connection failed: `+err.Error()+`"}`)
		w.WriteHeader(http.StatusOK)
		// Render failure modal
		renderTestResultModal(w, device.Name, "Connection Failed", "error", nil)
		return
	}

	// Update device status
	services.SyncDeviceInfo(id)

	w.Header().Set("HX-Trigger", `{"toast":"Connection successful"}`)
	renderTestResultModal(w, device.Name, "Online", "success", info)
}

// renderTestResultModal renders the test connection result in a modal
func renderTestResultModal(w http.ResponseWriter, deviceName string, status string, statusType string, info map[string]string) {
	statusBadge := fmt.Sprintf(`<span class="badge badge-%s">%s</span>`, statusType, status)

	// Build device info table rows
	infoRows := ""
	if info != nil {
		fields := []struct {
			Label string
			Key   string
		}{
			{"Device Name", "deviceName"},
			{"Device ID", "deviceID"},
			{"Device Type", "deviceType"},
			{"Serial Number", "serialNumber"},
			{"Firmware Version", "firmwareVersion"},
			{"Firmware Released Date", "firmwareReleasedDate"},
		}
		for _, f := range fields {
			value := info[f.Key]
			if value == "" {
				value = "-"
			}
			infoRows += fmt.Sprintf(`<tr><td style="padding: 0.5rem; font-weight: 600; color: var(--ink); white-space: nowrap;">%s</td><td style="padding: 0.5rem; color: var(--inkMuted);">%s</td></tr>`, f.Label, value)
		}
	}

	html := fmt.Sprintf(`
<div id="modal-overlay" class="modal-overlay" style="display:flex;">
    <div class="modal-content" style="max-width: 500px;">
        <div class="modal-header">
            <h2>Test Connection: %s</h2>
            <button class="btn btn-sm btn-ghost" onclick="document.getElementById('modal-overlay').remove()">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
        </div>
        <div class="modal-body">
            <div style="margin-bottom: 1rem;">Status: %s</div>
            <table style="width: 100%%; border-collapse: collapse;">
                <tbody>%s</tbody>
            </table>
        </div>
        <div class="modal-footer">
            <button class="btn btn-secondary" onclick="document.getElementById('modal-overlay').remove()">Close</button>
        </div>
    </div>
</div>`, deviceName, statusBadge, infoRows)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// AdminAudio renders the audio library page
func AdminAudio(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	var files []models.AudioFile
	var err error

	if query != "" {
		files, err = services.SearchAudioFiles(query)
	} else {
		files, err = services.GetAllAudioFiles()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RenderDashboard(w, r, "audio", files)
}

// AdminAudioUpload handles audio file upload
func AdminAudioUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseMultipartForm(32 << 20) // 32MB max
		file, header, err := r.FormFile("audio_file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		category := r.FormValue("category")
		if category == "" {
			category = "Custom"
		}

		// Ensure uploads directory exists
		uploadDir := "uploads/audio"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}

		// Save file to disk
		filePath := uploadDir + "/" + header.Filename
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Read audio metadata using ffprobe (duration, bitrate, sample rate)
		meta, _ := services.GetAudioMetadata(filePath)
		duration := 0
		durationStr := ""
		bitrate := 0
		sampleRate := 0
		if meta != nil {
			duration = meta.Duration
			durationStr = meta.DurationStr
			bitrate = meta.Bitrate
			sampleRate = meta.SampleRate
		}

		// Fallback if ffprobe is not available
		if duration == 0 {
			duration, _ = services.GetAudioDuration(filePath)
		}

		audioFile := &models.AudioFile{
			Name:        header.Filename,
			Category:    category,
			Duration:    duration,
			DurationStr: durationStr,
			FileSize:    header.Size,
			Bitrate:     bitrate,
			SampleRate:  sampleRate,
			FilePath:    filePath,
		}

		_, err = services.CreateAudioFile(audioFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("HX-Trigger", `{"toast":"Audio uploaded successfully","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	// If HTMX request, render only the modal content without layout
	if r.Header.Get("HX-Request") == "true" {
		user, _ := r.Context().Value("user").(*models.User)
		if user == nil {
			user = &models.User{Name: "Admin", Username: "admin"}
		}
		pageData := templates.PageData{
			Title: "Upload Audio",
			User:  user,
			Data:  nil,
		}
		templates.RenderPartial(w, "dashboard", "audio_upload", pageData)
		return
	}

	RenderDashboard(w, r, "audio_upload", nil)
}

// AdminAudioDelete handles audio file deletion
func AdminAudioDelete(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/audio/delete/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = services.DeleteAudioFile(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":"Audio deleted successfully","reload":"true"}`)
	w.WriteHeader(http.StatusOK)
}

// AdminSchedules renders the broadcast schedule page
func AdminSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := services.GetAllSchedules()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RenderDashboard(w, r, "schedules", schedules)
}

// AdminSchedulesCreate handles schedule creation
func AdminSchedulesCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		audioID, _ := strconv.Atoi(r.FormValue("audio_id"))
		deviceID, _ := strconv.Atoi(r.FormValue("device_id"))
		volume, _ := strconv.Atoi(r.FormValue("volume"))
		enabled := r.FormValue("enabled") == "on"

		schedule := &models.BroadcastSchedule{
			Name:         r.FormValue("name"),
			AudioID:      audioID,
			DeviceID:     deviceID,
			ScheduleType: r.FormValue("schedule_type"),
			BeginTime:    r.FormValue("begin_time"),
			EndTime:      r.FormValue("end_time"),
			Volume:       volume,
			Enabled:      enabled,
		}

		if schedule.ScheduleType == "weekly" {
			if dayOfWeekStr := r.FormValue("day_of_week"); dayOfWeekStr != "" {
				dayOfWeek, _ := strconv.Atoi(dayOfWeekStr)
				schedule.DayOfWeek = &dayOfWeek
			}
		}

		if schedule.ScheduleType == "specific_date" {
			date := r.FormValue("specific_date")
			schedule.SpecificDate = &date
		}

		_, err := services.CreateSchedule(schedule)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("HX-Trigger", `{"toast":"Schedule created successfully","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	devices, _ := services.GetAllDevices()
	audioFiles, _ := services.GetAllAudioFiles()
	data := map[string]interface{}{
		"Devices": devices,
		"Audio":   audioFiles,
	}

	// If HTMX request, render only the modal content without layout
	if r.Header.Get("HX-Request") == "true" {
		user, _ := r.Context().Value("user").(*models.User)
		if user == nil {
			user = &models.User{Name: "Admin", Username: "admin"}
		}
		pageData := templates.PageData{
			Title: "Add Schedule",
			User:  user,
			Data:  data,
		}
		templates.RenderPartial(w, "dashboard", "schedules_form", pageData)
		return
	}

	RenderDashboard(w, r, "schedules_form", data)
}

// AdminSchedulesEdit handles schedule edit

func AdminSchedulesEdit(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/schedules/edit/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method == "POST" {
		audioID, _ := strconv.Atoi(r.FormValue("audio_id"))
		deviceID, _ := strconv.Atoi(r.FormValue("device_id"))
		volume, _ := strconv.Atoi(r.FormValue("volume"))
		enabled := r.FormValue("enabled") == "on"

		schedule := &models.BroadcastSchedule{
			ID:           id,
			Name:         r.FormValue("name"),
			AudioID:      audioID,
			DeviceID:     deviceID,
			ScheduleType: r.FormValue("schedule_type"),
			BeginTime:    r.FormValue("begin_time"),
			EndTime:      r.FormValue("end_time"),
			Volume:       volume,
			Enabled:      enabled,
		}

		err := services.UpdateSchedule(schedule)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("HX-Trigger", `{"toast":"Schedule updated successfully","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	schedule, err := services.GetScheduleByID(id)
	if err != nil {
		http.Error(w, "Schedule not found", http.StatusNotFound)
		return
	}

	devices, _ := services.GetAllDevices()
	audioFiles, _ := services.GetAllAudioFiles()
	data := map[string]interface{}{
		"Schedule": schedule,
		"Devices":  devices,
		"Audio":    audioFiles,
	}

	// If HTMX request, render only the modal content without layout
	if r.Header.Get("HX-Request") == "true" {
		user, _ := r.Context().Value("user").(*models.User)
		if user == nil {
			user = &models.User{Name: "Admin", Username: "admin"}
		}
		pageData := templates.PageData{
			Title: "Edit Schedule",
			User:  user,
			Data:  data,
		}
		templates.RenderPartial(w, "dashboard", "schedules_form", pageData)
		return
	}

	RenderDashboard(w, r, "schedules_form", data)
}

// AdminSchedulesDelete handles schedule deletion
func AdminSchedulesDelete(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/schedules/delete/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = services.DeleteSchedule(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":"Schedule deleted successfully","reload":"true"}`)
	w.WriteHeader(http.StatusOK)
}

// AdminSchedulesSync handles syncing schedule to device
func AdminSchedulesSync(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/schedules/sync/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = services.SyncScheduleToDevice(id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"toast":"Sync failed: `+err.Error()+`"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("HX-Trigger", `{"toast":"Schedule synced to device successfully"}`)
	w.WriteHeader(http.StatusOK)
}

// AdminLogs renders the broadcast log page
func AdminLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := services.GetAllLogs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RenderDashboard(w, r, "logs", logs)
}

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
		prayerTimes, _ = services.GetUpcomingPrayerTimes(7)
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
				AudioID:  0,
				DeviceID: 0,
				Volume:   50,
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

// AdminPrayerSave handles prayer location save and auto-generates prayer times
func AdminPrayerSave(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		latitude, _ := strconv.ParseFloat(r.FormValue("latitude"), 64)
		longitude, _ := strconv.ParseFloat(r.FormValue("longitude"), 64)
		method, _ := strconv.Atoi(r.FormValue("method"))

		location := &models.PrayerLocation{
			Latitude:  latitude,
			Longitude: longitude,
			Timezone:  r.FormValue("timezone"),
			Method:    method,
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
			w.Header().Set("HX-Trigger", `{"toast":"Please set a location first"}`)
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
			w.Header().Set("HX-Trigger", `{"toast":"Failed to generate prayer times: `+err.Error()+`"}`)
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("HX-Trigger", `{"toast":"Prayer times generated for `+strconv.Itoa(days)+` days","reload":"true"}`)
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
				AudioID:  audioID,
				DeviceID: deviceID,
				Volume:   volume,
				Enabled:  enabled,
			}
			services.SavePrayerBroadcastConfig(cfg)
		}

		w.Header().Set("HX-Trigger", `{"toast":"Prayer broadcast settings saved","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// AdminPrayerCreateSchedules creates broadcast schedules from prayer times
func AdminPrayerCreateSchedules(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		location, err := services.GetPrayerLocation()
		if err != nil {
			w.Header().Set("HX-Trigger", `{"toast":"Please set a location first"}`)
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

		err = services.CreatePrayerSchedules(location, days)
		if err != nil {
			w.Header().Set("HX-Trigger", `{"toast":"Failed to create schedules: `+err.Error()+`"}`)
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("HX-Trigger", `{"toast":"Prayer schedules created for `+strconv.Itoa(days)+` days","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// AdminSettings renders the settings page
func AdminSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := services.GetAllSettings()
	if err != nil {
		settings = []models.Setting{}
	}
	RenderDashboard(w, r, "settings", settings)
}

// AdminSettingsSave handles settings save
func AdminSettingsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		for key, values := range r.Form {
			if len(values) > 0 {
				services.SetSetting(key, values[0], "")
			}
		}

		w.Header().Set("HX-Trigger", `{"toast":"Settings saved successfully","reload":"true"}`)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// AdminBroadcastNow handles manual broadcast
func AdminBroadcastNow(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		deviceID, _ := strconv.Atoi(r.FormValue("device_id"))
		audioID, _ := strconv.Atoi(r.FormValue("audio_id"))
		volume, _ := strconv.Atoi(r.FormValue("volume"))
		durationMinutes, _ := strconv.Atoi(r.FormValue("duration"))

		device, err := services.GetDeviceByID(deviceID)
		if err != nil {
			http.Error(w, "Device not found", http.StatusNotFound)
			return
		}

		audio, err := services.GetAudioFileByID(audioID)
		if err != nil {
			http.Error(w, "Audio not found", http.StatusNotFound)
			return
		}

		// Send broadcast to the Hikvision device
		err = services.BroadcastToDevice(device, audioID, volume, durationMinutes)
		if err != nil {
			// Log the failed broadcast
			log := &models.BroadcastLog{
				Time:       r.FormValue("time"),
				DeviceID:   sql.NullInt64{Int64: int64(deviceID), Valid: deviceID > 0},
				DeviceName: device.Name,
				AudioID:    sql.NullInt64{Int64: int64(audioID), Valid: audioID > 0},
				AudioName:  audio.Name,
				Result:     "failed",
				Status:     "error",
				Duration:   durationMinutes,
			}
			services.CreateLog(log)

			w.Header().Set("HX-Trigger", `{"toast":"Broadcast failed: `+err.Error()+`"}`)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log the successful broadcast
		log := &models.BroadcastLog{
			Time:       r.FormValue("time"),
			DeviceID:   sql.NullInt64{Int64: int64(deviceID), Valid: deviceID > 0},
			DeviceName: device.Name,
			AudioID:    sql.NullInt64{Int64: int64(audioID), Valid: audioID > 0},
			AudioName:  audio.Name,
			Result:     "success",
			Status:     "completed",
			Duration:   durationMinutes,
		}
		services.CreateLog(log)

		w.Header().Set("HX-Trigger", `{"toast":"Broadcast started successfully"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	devices, _ := services.GetAllDevices()
	audioFiles, _ := services.GetAllAudioFiles()
	data := map[string]interface{}{
		"Devices": devices,
		"Audio":   audioFiles,
	}
	RenderDashboard(w, r, "broadcast_now", data)
}
