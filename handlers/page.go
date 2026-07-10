package handlers

import (
	"ego/models"
	"ego/services"
	"ego/templates"
	"net/http"
)

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
