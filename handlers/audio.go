package handlers

import (
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
		devices, _ := services.GetAllDevices()
		pageData := templates.PageData{
			Title: "Upload Audio",
			User:  user,
			Data:  devices,
		}
		templates.RenderPartial(w, "dashboard", "audio_upload", pageData)
		return
	}

	devices, _ := services.GetAllDevices()
	RenderDashboard(w, r, "audio_upload", devices)
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
            <h2>Sync from Device</h2>
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

		synced, err := services.SyncAudioFromDevice(deviceID)
		if err != nil {
			setHXTriggerToast(w, "Sync failed: "+err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}

		setHXTriggerToast(w, fmt.Sprintf("Synced %d audio files from device", synced), true)
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: render sync form as modal
	devices, _ := services.GetAllDevices()
	renderAudioSyncModal(w, devices)
}

// AdminAudioSyncToDevice syncs audio files from local database to all enabled Hikvision devices.
// Deletes device audio that no longer exists in local DB.
func AdminAudioSyncToDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices, err := services.GetAllDevices()
	if err != nil {
		setHXTriggerToast(w, "Failed to get devices: "+err.Error())
		w.WriteHeader(http.StatusOK)
		return
	}

	var lastErr error
	totalDeleted := 0
	syncedDevices := 0
	for _, d := range devices {
		if !d.Enabled {
			continue
		}
		deleted, err := services.SyncAudioToDevice(d.ID)
		if err != nil {
			log.Printf("[AUDIO SYNC TO] Failed on device %s: %v", d.Name, err)
			lastErr = err
			continue
		}
		totalDeleted += deleted
		syncedDevices++
	}

	if syncedDevices > 0 {
		setHXTriggerToast(w, fmt.Sprintf("Synced to %d device(s): %d orphan audio(s) removed", syncedDevices, totalDeleted), true)
	} else if lastErr != nil {
		setHXTriggerToast(w, "Sync to device failed: "+lastErr.Error())
	} else {
		setHXTriggerToast(w, "No enabled devices available to sync")
	}
	w.WriteHeader(http.StatusOK)
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
