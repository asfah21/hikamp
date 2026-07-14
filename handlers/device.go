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
