package handlers

import (
	"ego/models"
	"ego/services"
	"ego/templates"
	"net/http"
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

// RenderDashboard renders a dashboard page with sidebar layout
func RenderDashboard(w http.ResponseWriter, r *http.Request, page string, data interface{}) {
	// Get user from session
	user, _ := r.Context().Value("user").(*models.User)
	if user == nil {
		user = &models.User{Name: "Admin", Username: "admin"}
	}

	// Get settings for sidebar (graceful fallback)
	companyName := "Hikvision Broadcast"
	if settings, err := services.GetAllSettings(); err == nil {
		for _, s := range settings {
			if s.Key == "company_name" && s.Value != "" {
				companyName = s.Value
			}
		}
	}

	// Get current path for HTMX reload
	currentPath := r.URL.Path
	if currentPath == "" || currentPath == "/admin" {
		currentPath = "/admin/dashboard"
	}

	pageData := templates.PageData{
		Title:       getPageTitle(page),
		User:        user,
		Data:        data,
		CompanyName: companyName,
		Page:        page,
		CurrentPath: currentPath,
	}

	// If HTMX request (for page reload), render only the content partial without layout
	if r.Header.Get("HX-Request") == "true" {
		templates.RenderPartial(w, "dashboard", page, pageData)
		return
	}

	templates.Render(w, "dashboard", page, pageData)
}

// getPageTitle returns the title for a given page
func getPageTitle(page string) string {
	titles := map[string]string{
		"dashboard":        "Dashboard",
		"devices":          "Devices",
		"devices_form":     "Device",
		"audio":            "Audio Library",
		"audio_upload":     "Upload Audio",
		"schedules":        "Broadcast Schedules",
		"schedules_form":   "Schedule",
		"logs":             "Broadcast Logs",
		"prayer":           "Prayer Schedule",
		"prayer_setting":   "Prayer Settings",
		"prayer_time":      "Prayer Times",
		"prayer_broadcast": "Prayer Broadcast",
		"settings":         "Settings",
		"broadcast_now":    "Manual Broadcast",
	}
	if title, ok := titles[page]; ok {
		return title
	}
	return "Dashboard"
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

// AdminLogs renders the broadcast log page
func AdminLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := services.GetAllLogs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RenderDashboard(w, r, "logs", logs)
}
