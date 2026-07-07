package handlers

import (
	"database/sql"
	"ego/internal/hikvision"
	"ego/models"
	"ego/services"
	"ego/templates"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// jsonEscape escapes a string for safe embedding in a JSON string value.
// This prevents "Bad control character" errors when error messages contain
// quotes, newlines, or other special characters.
func jsonEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// setHXTriggerToast sets the HX-Trigger header with a JSON toast message.
// The message is automatically JSON-escaped to prevent invalid JSON errors.
// If reload is true, adds "reload":"true" to trigger a page reload.
func setHXTriggerToast(w http.ResponseWriter, message string, reload ...bool) {
	escaped := jsonEscape(message)
	doReload := len(reload) > 0 && reload[0]
	if doReload {
		w.Header().Set("HX-Trigger", `{"toast":"`+escaped+`","reload":"true"}`)
	} else {
		w.Header().Set("HX-Trigger", `{"toast":"`+escaped+`"}`)
	}
}

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

		setHXTriggerToast(w, "Device created successfully", true)
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

		setHXTriggerToast(w, "Device updated successfully", true)
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

	setHXTriggerToast(w, "Device deleted successfully", true)
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
		setHXTriggerToast(w, "Connection failed: "+err.Error())
		w.WriteHeader(http.StatusOK)
		// Render failure modal
		renderTestResultModal(w, device.Name, "Connection Failed", "error", nil)
		return
	}

	// Update device status
	services.SyncDeviceInfo(id)

	setHXTriggerToast(w, "Connection successful")
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

// renderAudioSyncModal renders the sync audio modal
func renderAudioSyncModal(w http.ResponseWriter, devices []models.Device) {
	deviceOptions := ""
	for _, d := range devices {
		deviceOptions += fmt.Sprintf(`<option value="%d">%s (%s)</option>`, d.ID, d.Name, d.IPAddress)
	}
	if len(devices) == 0 {
		deviceOptions = `<option value="" disabled>No devices available</option>`
	}

	html := fmt.Sprintf(`
<div id="modal-overlay" class="modal-overlay" style="display:flex;">
    <div class="modal-content" style="max-width: 500px;">
        <div class="modal-header">
            <h2>Sync Audio from Device</h2>
            <button class="btn btn-sm btn-ghost" onclick="document.getElementById('modal-overlay').remove()">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
        </div>
        <div class="modal-body">
            <p style="margin-bottom: 1rem; color: var(--inkMuted);">
                Fetch all audio files from a Hikvision device and sync them to the local database.
                Existing audio files will be updated, new ones will be added.
            </p>
            <form id="sync-form" hx-post="/admin/audio/sync" hx-target="#modal-container" hx-swap="innerHTML" hx-indicator="#sync-spinner">
                <div class="form-group">
                    <label class="form-label" for="device_id">Device</label>
                    <select class="input-field" id="device_id" name="device_id" required>
                        <option value="">Select a device...</option>
                        %s
                    </select>
                </div>
                <div class="form-actions" style="margin-top: 1.5rem; display: flex; gap: 0.5rem; align-items: center;">
                    <button type="submit" class="btn btn-primary" id="sync-btn">
                        <span id="sync-spinner" class="htmx-indicator" style="display: inline-flex; align-items: center;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 0.5rem; animation: spin 1s linear infinite;"><path d="M21 2v6H3M3 2v6h18"/><path d="M21 12v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6"/><line x1="12" y1="12" x2="12" y2="18"/></svg>
                            Syncing...
                        </span>
                        <span class="htmx-indicator-hidden">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 0.5rem;"><path d="M21 2v6H3M3 2v6h18"/><path d="M21 12v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6"/><line x1="12" y1="12" x2="12" y2="18"/></svg>
                            Sync from Device
                        </span>
                    </button>
                    <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal-overlay').remove()">Cancel</button>
                </div>
            </form>
        </div>
    </div>
</div>`, deviceOptions)

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

// AdminAudioUpload handles audio file upload directly to Hikvision device.
// No local file storage — audio is uploaded directly to the device.
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

		deviceIDStr := r.FormValue("device_id")
		deviceID := 0
		if deviceIDStr != "" {
			deviceID, _ = strconv.Atoi(deviceIDStr)
		}

		if deviceID == 0 {
			setHXTriggerToast(w, "Please select a device to upload to")
			w.WriteHeader(http.StatusOK)
			return
		}

		device, err := services.GetDeviceByID(deviceID)
		if err != nil || device == nil {
			http.Error(w, "Device not found", http.StatusNotFound)
			return
		}

		// Upload directly to Hikvision device
		client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)
		audioID, err := client.UploadAudio(file, header.Filename)
		if err != nil {
			setHXTriggerToast(w, "Failed to upload audio to device: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		// After upload, search audio on device to get metadata (duration, path)
		audioInfo, _ := client.SearchAudioByID(audioID)

		hikvisionAudioID := audioID
		audioFile := &models.AudioFile{
			Name:             header.Filename,
			Category:         category,
			Duration:         0,
			DurationStr:      "",
			FileSize:         header.Size,
			HikvisionAudioID: &hikvisionAudioID,
			HikvisionPath:    nil,
			DeviceID:         &deviceID,
		}

		if audioInfo != nil {
			audioFile.Duration = audioInfo.Duration
			audioFile.DurationStr = audioInfo.DurationStr
			audioFile.HikvisionPath = &audioInfo.HikvisionPath
		}

		_, err = services.CreateAudioFile(audioFile)
		if err != nil {
			setHXTriggerToast(w, "Audio uploaded to device but failed to save to database: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		setHXTriggerToast(w, "Audio uploaded to device successfully", true)
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

// AdminAudioSync syncs audio files from Hikvision device to local database.
// Fetches the audio list from the device and upserts into audio_files table.
func AdminAudioSync(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		deviceIDStr := r.FormValue("device_id")
		deviceID := 0
		if deviceIDStr != "" {
			deviceID, _ = strconv.Atoi(deviceIDStr)
		}

		if deviceID == 0 {
			setHXTriggerToast(w, "Please select a device to sync from")
			w.WriteHeader(http.StatusOK)
			return
		}

		device, err := services.GetDeviceByID(deviceID)
		if err != nil || device == nil {
			http.Error(w, "Device not found", http.StatusNotFound)
			return
		}

		client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)
		audioList, err := client.SearchAudio()
		if err != nil {
			setHXTriggerToast(w, "Failed to fetch audio from device: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		synced := 0
		for _, audio := range audioList {
			hikvisionAudioID := audio.CustomAudioID
			hikvisionPath := audio.HikvisionPath
			audioFile := &models.AudioFile{
				Name:             audio.CustomAudioName,
				Category:         "Custom",
				Duration:         audio.Duration,
				DurationStr:      audio.DurationStr,
				FileSize:         int64(audio.AudioFileSize),
				HikvisionAudioID: &hikvisionAudioID,
				HikvisionPath:    &hikvisionPath,
				DeviceID:         &deviceID,
			}
			_, err := services.UpsertAudioFileByHikvisionID(audioFile)
			if err != nil {
				log.Printf("[AUDIO SYNC] Failed to upsert audio '%s': %v", audio.CustomAudioName, err)
				continue
			}
			synced++
		}

		setHXTriggerToast(w, fmt.Sprintf("Synced %d audio files from device", synced), true)
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: render sync form as modal
	devices, _ := services.GetAllDevices()
	renderAudioSyncModal(w, devices)
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

	setHXTriggerToast(w, "Audio deleted successfully", true)
	w.WriteHeader(http.StatusOK)
}

// AdminAudioDownload streams audio from Hikvision device to the browser.
// Uses the device's HTTP endpoint to fetch the audio file and proxy it.
func AdminAudioDownload(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/audio/download/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	audioFile, err := services.GetAudioFileByID(id)
	if err != nil {
		http.Error(w, "Audio not found", http.StatusNotFound)
		return
	}

	if audioFile.DeviceID == nil || *audioFile.DeviceID == 0 {
		http.Error(w, "Audio has no associated device", http.StatusNotFound)
		return
	}

	device, err := services.GetDeviceByID(*audioFile.DeviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Build the audio URL on the Hikvision device
	// Hikvision serves audio files via HTTP at the customAudioPath
	audioURL := fmt.Sprintf("http://%s:%d%s", device.IPAddress, device.Port, *audioFile.HikvisionPath)

	// Create a digest client and fetch the audio
	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)
	req, err := http.NewRequest("GET", audioURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := client.DigestClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch audio from device: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		http.Error(w, "Device returned error", http.StatusBadGateway)
		return
	}

	// Set headers for download/playback
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, audioFile.Name))

	// Stream the audio data
	io.Copy(w, resp.Body)
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

		setHXTriggerToast(w, "Schedule created successfully", true)
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

		setHXTriggerToast(w, "Schedule updated successfully", true)
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

	setHXTriggerToast(w, "Schedule deleted successfully", true)
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
		setHXTriggerToast(w, "Sync failed: "+err.Error())
		w.WriteHeader(http.StatusOK)
		return
	}

	setHXTriggerToast(w, "Schedule synced to device successfully", true)
	w.WriteHeader(http.StatusOK)
}

// AdminSchedulesSyncAll handles syncing all schedules to their respective devices
func AdminSchedulesSyncAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	success, fail, err := services.SyncAllSchedulesToDevice()
	if err != nil {
		setHXTriggerToast(w, fmt.Sprintf("Sync completed: %d success, %d failed. Last error: %s", success, fail, err.Error()), true)
	} else {
		setHXTriggerToast(w, fmt.Sprintf("All %d schedules synced to device successfully", success), true)
	}

	w.WriteHeader(http.StatusOK)
}

// AdminSchedulesSyncFrom handles sync-from-device (GET = modal, POST = execute)
func AdminSchedulesSyncFrom(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		deviceIDStr := r.FormValue("device_id")
		deviceID := 0
		if deviceIDStr != "" {
			deviceID, _ = strconv.Atoi(deviceIDStr)
		}

		if deviceID == 0 {
			setHXTriggerToast(w, "Please select a device to sync from")
			w.WriteHeader(http.StatusOK)
			return
		}

		synced, err := services.SyncSchedulesFromDevice(deviceID)
		if err != nil {
			setHXTriggerToast(w, "Sync failed: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		setHXTriggerToast(w, fmt.Sprintf("Imported %d schedules from device", synced), true)
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: render sync form as modal
	devices, _ := services.GetAllDevices()
	renderScheduleSyncFromModal(w, devices)
}

// renderScheduleSyncFromModal renders the sync-from-device modal for schedules
func renderScheduleSyncFromModal(w http.ResponseWriter, devices []models.Device) {
	deviceOptions := ""
	for _, d := range devices {
		deviceOptions += fmt.Sprintf(`<option value="%d">%s (%s)</option>`, d.ID, d.Name, d.IPAddress)
	}
	if len(devices) == 0 {
		deviceOptions = `<option value="" disabled>No devices available</option>`
	}

	html := fmt.Sprintf(`
<div id="modal-overlay" class="modal-overlay" style="display:flex;">
    <div class="modal-content" style="max-width: 500px;">
        <div class="modal-header">
            <h2>Sync Schedules from Device</h2>
            <button class="btn btn-sm btn-ghost" onclick="document.getElementById('modal-overlay').remove()">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
        </div>
        <div class="modal-body">
            <p style="margin-bottom: 1rem; color: var(--inkMuted);">
                Fetch all broadcast schedules from a Hikvision device and import them into the local database.
            </p>
            <form id="sync-form" hx-post="/admin/schedules/sync-from" hx-target="#modal-container" hx-swap="innerHTML" hx-indicator="#sync-spinner">
                <div class="form-group">
                    <label class="form-label" for="device_id">Device</label>
                    <select class="input-field" id="device_id" name="device_id" required>
                        <option value="">Select a device...</option>
                        %s
                    </select>
                </div>
                <div class="form-actions" style="margin-top: 1.5rem; display: flex; gap: 0.5rem; align-items: center;">
                    <button type="submit" class="btn btn-primary" id="sync-btn">
                        <span id="sync-spinner" class="htmx-indicator" style="display: inline-flex; align-items: center;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 0.5rem; animation: spin 1s linear infinite;"><path d="M21 2v6H3M3 2v6h18"/><path d="M21 12v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6"/><line x1="12" y1="12" x2="12" y2="18"/></svg>
                            Syncing...
                        </span>
                        <span class="htmx-indicator-hidden">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 0.5rem;"><path d="M21 2v6H3M3 2v6h18"/><path d="M21 12v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6"/><line x1="12" y1="12" x2="12" y2="18"/></svg>
                            Sync from Device
                        </span>
                    </button>
                    <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal-overlay').remove()">Cancel</button>
                </div>
            </form>
        </div>
    </div>
</div>`, deviceOptions)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
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
				AudioID:  audioID,
				DeviceID: deviceID,
				Volume:   volume,
				Enabled:  enabled,
			}
			services.SavePrayerBroadcastConfig(cfg)
		}

		// After saving broadcast configs, sync schedules to devices
		location, err := services.GetPrayerLocation()
		if err == nil && location != nil {
			// Generate schedules for the next 30 days and sync to devices
			go func() {
				services.CreatePrayerSchedules(location, 30)
				// Sync all enabled schedules to their devices
				schedules, _ := services.GetAllSchedules()
				for _, s := range schedules {
					if s.Enabled {
						services.SyncScheduleToDevice(s.ID)
					}
				}
			}()
		}

		setHXTriggerToast(w, "Prayer broadcast settings saved and synced to devices", true)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// AdminPrayerCreateSchedules creates broadcast schedules from prayer times
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

		err = services.CreatePrayerSchedules(location, days)
		if err != nil {
			setHXTriggerToast(w, "Failed to create schedules: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		// Sync all enabled schedules to their devices
		go func() {
			schedules, _ := services.GetAllSchedules()
			for _, s := range schedules {
				if s.Enabled {
					services.SyncScheduleToDevice(s.ID)
				}
			}
		}()

		setHXTriggerToast(w, "Prayer schedules created for "+strconv.Itoa(days)+" days and synced to devices", true)
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

		setHXTriggerToast(w, "Settings saved successfully", true)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// AdminStopBroadcast handles stopping all broadcasts on a device
func AdminStopBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		deviceID, _ := strconv.Atoi(r.FormValue("device_id"))

		if deviceID == 0 {
			// If no device specified, stop on all devices
			devices, err := services.GetAllDevices()
			if err != nil {
				setHXTriggerToast(w, "Failed to get devices: "+err.Error())
				w.WriteHeader(http.StatusOK)
				return
			}

			var lastErr error
			stopped := 0
			for _, d := range devices {
				if !d.Enabled {
					continue
				}
				err := services.StopBroadcastOnDevice(d.ID)
				if err != nil {
					log.Printf("[STOP BROADCAST] Failed on device %s: %v", d.Name, err)
					lastErr = err
					continue
				}
				stopped++
			}

			if stopped > 0 {
				setHXTriggerToast(w, fmt.Sprintf("Broadcast stopped on %d device(s)", stopped), true)
			} else if lastErr != nil {
				setHXTriggerToast(w, "Failed to stop broadcast: "+lastErr.Error())
			} else {
				setHXTriggerToast(w, "No active broadcasts found")
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		// Stop on specific device
		err := services.StopBroadcastOnDevice(deviceID)
		if err != nil {
			setHXTriggerToast(w, "Failed to stop broadcast: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		setHXTriggerToast(w, "Broadcast stopped successfully", true)
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: render stop broadcast form as modal
	devices, _ := services.GetAllDevices()
	renderStopBroadcastModal(w, devices)
}

// renderStopBroadcastModal renders the stop broadcast confirmation modal
func renderStopBroadcastModal(w http.ResponseWriter, devices []models.Device) {
	deviceOptions := ""
	for _, d := range devices {
		deviceOptions += fmt.Sprintf(`<option value="%d">%s (%s)</option>`, d.ID, d.Name, d.IPAddress)
	}
	if len(devices) == 0 {
		deviceOptions = `<option value="" disabled>No devices available</option>`
	}

	html := fmt.Sprintf(`
<div id="modal-overlay" class="modal-overlay" style="display:flex;">
    <div class="modal-content" style="max-width: 450px;">
        <div class="modal-header">
            <h2>Stop Broadcast</h2>
            <button class="btn btn-sm btn-ghost" onclick="document.getElementById('modal-overlay').remove()">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
        </div>
        <div class="modal-body">
            <p style="margin-bottom: 1rem; color: var(--inkMuted);">
                Stop all active broadcasts on a device. This will disable active schedules
                without deleting them. Select a specific device or stop all devices.
            </p>
            <form id="stop-form" hx-post="/admin/stop-broadcast" hx-target="#modal-container" hx-swap="innerHTML" hx-indicator="#stop-spinner">
                <div class="form-group">
                    <label class="form-label" for="device_id">Device</label>
                    <select class="input-field" id="device_id" name="device_id">
                        <option value="0">All Devices</option>
                        %s
                    </select>
                </div>
                <div class="form-actions" style="margin-top: 1.5rem; display: flex; gap: 0.5rem; align-items: center;">
                    <button type="submit" class="btn btn-danger" id="stop-btn">
                        <span id="stop-spinner" class="htmx-indicator" style="display: inline-flex; align-items: center;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 0.5rem; animation: spin 1s linear infinite;"><path d="M21 2v6H3M3 2v6h18"/><path d="M21 12v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6"/><line x1="12" y1="12" x2="12" y2="18"/></svg>
                            Stopping...
                        </span>
                        <span class="htmx-indicator-hidden">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 0.5rem;"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>
                            Stop Broadcast
                        </span>
                    </button>
                    <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal-overlay').remove()">Cancel</button>
                </div>
            </form>
        </div>
    </div>
</div>`, deviceOptions)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// AdminBroadcastNow handles manual broadcast
func AdminBroadcastNow(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		deviceID, _ := strconv.Atoi(r.FormValue("device_id"))
		audioID, _ := strconv.Atoi(r.FormValue("audio_id"))
		volume, _ := strconv.Atoi(r.FormValue("volume"))

		// Parse duration from hours:minutes:seconds picker
		durationHours, _ := strconv.Atoi(r.FormValue("duration_hours"))
		durationMinutes, _ := strconv.Atoi(r.FormValue("duration_minutes"))
		durationSeconds, _ := strconv.Atoi(r.FormValue("duration_seconds"))
		totalMinutes := durationHours*60 + durationMinutes + durationSeconds/60
		if durationSeconds%60 > 0 {
			totalMinutes++
		}

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
		err = services.BroadcastToDevice(device, audioID, volume, totalMinutes)
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
				Duration:   totalMinutes,
			}
			services.CreateLog(log)

			setHXTriggerToast(w, "Broadcast failed: "+err.Error())
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
			Duration:   totalMinutes,
		}
		services.CreateLog(log)

		setHXTriggerToast(w, "Broadcast started successfully", true)
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
