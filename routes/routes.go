package routes

import (
	"ego/handlers"
	"net/http"
)

// Register registers all HTTP routes and returns the handler
func Register() http.Handler {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	// Public pages
	mux.HandleFunc("/", handlers.HandlePublic)

	// Auth pages
	mux.HandleFunc("/login", handlers.HandleLogin)
	mux.HandleFunc("/logout", handlers.HandleLogout)

	// Admin routes (protected)
	mux.HandleFunc("/admin/dashboard", handlers.RequireAuth(handlers.AdminDashboard))
	mux.HandleFunc("/admin/devices", handlers.RequireAuth(handlers.AdminDevices))
	mux.HandleFunc("/admin/devices/create", handlers.RequireAuth(handlers.AdminDevicesCreate))
	mux.HandleFunc("/admin/devices/edit/", handlers.RequireAuth(handlers.AdminDevicesEdit))
	mux.HandleFunc("/admin/devices/delete/", handlers.RequireAuth(handlers.AdminDevicesDelete))
	mux.HandleFunc("/admin/devices/test/", handlers.RequireAuth(handlers.AdminDevicesTestConnection))
	mux.HandleFunc("/admin/audio", handlers.RequireAuth(handlers.AdminAudio))
	mux.HandleFunc("/admin/audio/upload", handlers.RequireAuth(handlers.AdminAudioUpload))
	mux.HandleFunc("/admin/audio/sync", handlers.RequireAuth(handlers.AdminAudioSync))
	mux.HandleFunc("/admin/audio/sync-to-device", handlers.RequireAuth(handlers.AdminAudioSyncToDevice))
	mux.HandleFunc("/admin/audio/delete/", handlers.RequireAuth(handlers.AdminAudioDelete))
	mux.HandleFunc("/admin/audio/download/", handlers.RequireAuth(handlers.AdminAudioDownload))

	mux.HandleFunc("/admin/schedules", handlers.RequireAuth(handlers.AdminSchedules))
	mux.HandleFunc("/admin/schedules/create", handlers.RequireAuth(handlers.AdminSchedulesCreate))
	mux.HandleFunc("/admin/schedules/edit/", handlers.RequireAuth(handlers.AdminSchedulesEdit))
	mux.HandleFunc("/admin/schedules/delete/", handlers.RequireAuth(handlers.AdminSchedulesDelete))
	mux.HandleFunc("/admin/schedules/sync/", handlers.RequireAuth(handlers.AdminSchedulesSync))
	mux.HandleFunc("/admin/schedules/sync-all", handlers.RequireAuth(handlers.AdminSchedulesSyncAll))
	mux.HandleFunc("/admin/schedules/sync-from", handlers.RequireAuth(handlers.AdminSchedulesSyncFrom))
	mux.HandleFunc("/admin/logs", handlers.RequireAuth(handlers.AdminLogs))
	// Prayer routes - 3 submenu pages
	mux.HandleFunc("/admin/prayer", handlers.RequireAuth(handlers.AdminPrayerSetting))
	mux.HandleFunc("/admin/prayer/setting", handlers.RequireAuth(handlers.AdminPrayerSetting))
	mux.HandleFunc("/admin/prayer/time", handlers.RequireAuth(handlers.AdminPrayerTime))
	mux.HandleFunc("/admin/prayer/broadcast", handlers.RequireAuth(handlers.AdminPrayerBroadcast))
	// Prayer action routes
	mux.HandleFunc("/admin/prayer/save", handlers.RequireAuth(handlers.AdminPrayerSave))
	mux.HandleFunc("/admin/prayer/generate", handlers.RequireAuth(handlers.AdminPrayerGenerate))
	mux.HandleFunc("/admin/prayer/broadcast/save", handlers.RequireAuth(handlers.AdminPrayerBroadcastSave))
	mux.HandleFunc("/admin/prayer/create-schedules", handlers.RequireAuth(handlers.AdminPrayerCreateSchedules))
	mux.HandleFunc("/admin/settings", handlers.RequireAuth(handlers.AdminSettings))
	mux.HandleFunc("/admin/settings/save", handlers.RequireAuth(handlers.AdminSettingsSave))
	mux.HandleFunc("/admin/broadcast-now", handlers.RequireAuth(handlers.AdminBroadcastNow))
	mux.HandleFunc("/admin/stop-broadcast", handlers.RequireAuth(handlers.AdminStopBroadcast))

	// Redirect /admin to /admin/dashboard
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	})

	return mux
}
