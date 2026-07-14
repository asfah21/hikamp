package handlers

import (
	"net/http"
)

// Router sets up all HTTP routes
func Router() http.Handler {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	// Public pages
	mux.HandleFunc("/", HandlePublic)

	// Auth pages
	mux.HandleFunc("/login", HandleLogin)
	mux.HandleFunc("/logout", HandleLogout)

	// Admin routes (protected)
	mux.HandleFunc("/admin/dashboard", RequireAuth(AdminDashboard))
	mux.HandleFunc("/admin/devices", RequireAuth(AdminDevices))
	mux.HandleFunc("/admin/devices/create", RequireAuth(AdminDevicesCreate))
	mux.HandleFunc("/admin/devices/edit/", RequireAuth(AdminDevicesEdit))
	mux.HandleFunc("/admin/devices/delete/", RequireAuth(AdminDevicesDelete))
	mux.HandleFunc("/admin/devices/test/", RequireAuth(AdminDevicesTestConnection))
	mux.HandleFunc("/admin/audio", RequireAuth(AdminAudio))
	mux.HandleFunc("/admin/audio/upload", RequireAuth(AdminAudioUpload))
	mux.HandleFunc("/admin/audio/sync", RequireAuth(AdminAudioSync))
	mux.HandleFunc("/admin/audio/sync-to-device", RequireAuth(AdminAudioSyncToDevice))
	mux.HandleFunc("/admin/audio/delete/", RequireAuth(AdminAudioDelete))
	mux.HandleFunc("/admin/audio/download/", RequireAuth(AdminAudioDownload))

	mux.HandleFunc("/admin/schedules", RequireAuth(AdminSchedules))
	mux.HandleFunc("/admin/schedules/create", RequireAuth(AdminSchedulesCreate))
	mux.HandleFunc("/admin/schedules/edit/", RequireAuth(AdminSchedulesEdit))
	mux.HandleFunc("/admin/schedules/delete/", RequireAuth(AdminSchedulesDelete))
	mux.HandleFunc("/admin/schedules/sync/", RequireAuth(AdminSchedulesSync))
	mux.HandleFunc("/admin/schedules/sync-all", RequireAuth(AdminSchedulesSyncAll))
	mux.HandleFunc("/admin/schedules/sync-from", RequireAuth(AdminSchedulesSyncFrom))
	mux.HandleFunc("/admin/logs", RequireAuth(AdminLogs))
	// Prayer routes - 3 submenu pages
	mux.HandleFunc("/admin/prayer", RequireAuth(AdminPrayerSetting))
	mux.HandleFunc("/admin/prayer/setting", RequireAuth(AdminPrayerSetting))
	mux.HandleFunc("/admin/prayer/time", RequireAuth(AdminPrayerTime))
	mux.HandleFunc("/admin/prayer/broadcast", RequireAuth(AdminPrayerBroadcast))
	// Prayer action routes
	mux.HandleFunc("/admin/prayer/save", RequireAuth(AdminPrayerSave))
	mux.HandleFunc("/admin/prayer/generate", RequireAuth(AdminPrayerGenerate))
	mux.HandleFunc("/admin/prayer/broadcast/save", RequireAuth(AdminPrayerBroadcastSave))
	mux.HandleFunc("/admin/prayer/create-schedules", RequireAuth(AdminPrayerCreateSchedules))
	mux.HandleFunc("/admin/settings", RequireAuth(AdminSettings))
	mux.HandleFunc("/admin/settings/save", RequireAuth(AdminSettingsSave))
	mux.HandleFunc("/admin/broadcast-now", RequireAuth(AdminBroadcastNow))
	mux.HandleFunc("/admin/stop-broadcast", RequireAuth(AdminStopBroadcast))

	// Redirect /admin to /admin/dashboard
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	})

	return mux
}
