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

	// Get settings for sidebar
	settings, _ := services.GetAllSettings()
	companyName := "Hikvision Broadcast"
	for _, s := range settings {
		if s.Key == "company_name" && s.Value != "" {
			companyName = s.Value
		}
	}

	pageData := templates.PageData{
		Title:       getPageTitle(page),
		User:        user,
		Data:        data,
		CompanyName: companyName,
		Page:        page,
	}

	templates.Render(w, "dashboard", page, pageData)
}

// getPageTitle returns the title for a given page
func getPageTitle(page string) string {
	titles := map[string]string{
		"dashboard":      "Dashboard",
		"devices":        "Devices",
		"devices_form":   "Device",
		"audio":          "Audio Library",
		"audio_upload":   "Upload Audio",
		"schedules":      "Broadcast Schedules",
		"schedules_form": "Schedule",
		"logs":           "Broadcast Logs",
		"prayer":         "Prayer Schedule",
		"settings":       "Settings",
		"broadcast_now":  "Manual Broadcast",
	}
	if title, ok := titles[page]; ok {
		return title
	}
	return "Dashboard"
}
