package handlers

import (
	"ego/models"
	"ego/services"
	"encoding/json"
	"fmt"
	"net/http"
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
		fmt.Fprintf(w, `<div class="badge badge-error">Connection Failed</div>`)
		return
	}

	// Update device status
	services.SyncDeviceInfo(id)

	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	w.Header().Set("HX-Trigger", `{"toast":"Connection successful"}`)
	fmt.Fprintf(w, `<div class="badge badge-success">Online</div><pre class="mt-2 text-xs">%s</pre>`, string(infoJSON))
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

		audioFile := &models.AudioFile{
			Name:     header.Filename,
			Category: category,
			FileSize: header.Size,
			FilePath: "uploads/audio/" + header.Filename,
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

// AdminPrayer renders the prayer schedule page
func AdminPrayer(w http.ResponseWriter, r *http.Request) {
	location, err := services.GetPrayerLocation()
	if err != nil {
		location = nil
	}

	data := map[string]interface{}{
		"Location": location,
	}
	RenderDashboard(w, r, "prayer", data)
}

// AdminPrayerSave handles prayer location save
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

		w.Header().Set("HX-Trigger", `{"toast":"Prayer location saved successfully","reload":"true"}`)
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

		// Log the broadcast
		log := &models.BroadcastLog{
			Time:       r.FormValue("time"),
			DeviceID:   deviceID,
			DeviceName: device.Name,
			AudioID:    audioID,
			AudioName:  audio.Name,
			Result:     "success",
			Status:     "completed",
			Duration:   volume,
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
