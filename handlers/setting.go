package handlers

import (
	"database/sql"
	"ego/models"
	"ego/services"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

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

// AdminBroadcastNow handles manual broadcast.
// Creates a schedule in the database (like prayer broadcast) then syncs all
// schedules to all devices — same as clicking "Sync All to Device".
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

		// Calculate begin_time and end_time for the schedule
		// Add 60 minutes + 2 seconds to compensate for timezone offset differences
		// between the server and the Hikvision device (same as BroadcastNowWithTimezone).
		now := time.Now()
		beginTime := now.Add(60*time.Minute + 2*time.Second).Format("15:04:05")
		endTime := now.Add(60*time.Minute + 2*time.Second + time.Duration(totalMinutes)*time.Minute).Format("15:04:05")

		// Create a schedule in the database (same pattern as prayer broadcast)
		schedule := &models.BroadcastSchedule{
			Name:         "Broadcast Now: " + audio.Name,
			ScheduleType: "daily",
			Enabled:      true,
			Entries: []models.ScheduleEntry{
				{
					AudioID:   audioID,
					BeginTime: beginTime,
					EndTime:   endTime,
					Volume:    volume,
				},
			},
			Devices: []models.ScheduleDevice{
				{
					DeviceID: deviceID,
				},
			},
		}

		_, err = services.CreateSchedule(schedule)
		if err != nil {
			setHXTriggerToast(w, "Failed to create schedule: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		// Sync all schedules to all devices (same as clicking "Sync All to Device")
		success, fail, syncErr := services.SyncAllSchedulesToDevice()

		nowStr := time.Now().Format("2006-01-02 15:04:05")
		if syncErr != nil || fail > 0 {
			// Log the failed broadcast
			log := &models.BroadcastLog{
				Time:       nowStr,
				DeviceID:   sql.NullInt64{Int64: int64(deviceID), Valid: deviceID > 0},
				DeviceName: device.Name,
				AudioID:    sql.NullInt64{Int64: int64(audioID), Valid: audioID > 0},
				AudioName:  audio.Name,
				Result:     "failed",
				Status:     "error",
				Duration:   totalMinutes,
			}
			services.CreateLog(log)

			msg := "Broadcast schedule created but sync had issues"
			if syncErr != nil {
				msg += ": " + syncErr.Error()
			}
			setHXTriggerToast(w, msg, true)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log the successful broadcast
		log := &models.BroadcastLog{
			Time:       nowStr,
			DeviceID:   sql.NullInt64{Int64: int64(deviceID), Valid: deviceID > 0},
			DeviceName: device.Name,
			AudioID:    sql.NullInt64{Int64: int64(audioID), Valid: audioID > 0},
			AudioName:  audio.Name,
			Result:     "success",
			Status:     "completed",
			Duration:   totalMinutes,
		}
		services.CreateLog(log)

		setHXTriggerToast(w, fmt.Sprintf("Broadcast started: %d schedule(s) synced to device(s)", success), true)
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
