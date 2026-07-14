package handlers

import (
	"ego/models"
	"ego/services"
	"ego/templates"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

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
