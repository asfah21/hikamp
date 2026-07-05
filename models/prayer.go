package models

// PrayerLocation stores location data for prayer time calculation
type PrayerLocation struct {
	ID        int     `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
	Method    int     `json:"method"` // Calculation method
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// PrayerTime represents a single prayer time
type PrayerTime struct {
	ID         int    `json:"id"`
	Date       string `json:"date"`
	Fajr       string `json:"fajr"`
	Dhuhr      string `json:"dhuhr"`
	Asr        string `json:"asr"`
	Maghrib    string `json:"maghrib"`
	Isha       string `json:"isha"`
	LocationID int    `json:"location_id"`
	CreatedAt  string `json:"created_at"`
}

// PrayerBroadcastConfig stores audio & device mapping for each prayer time
type PrayerBroadcastConfig struct {
	ID       int    `json:"id"`
	Prayer   string `json:"prayer"` // fajr, dhuhr, asr, maghrib, isha
	AudioID  int    `json:"audio_id"`
	DeviceID int    `json:"device_id"`
	Volume   int    `json:"volume"`
	Enabled  bool   `json:"enabled"`
}

// PrayerNames maps prayer keys to display names
var PrayerNames = map[string]string{
	"fajr":    "Fajr",
	"dhuhr":   "Dhuhr",
	"asr":     "Asr",
	"maghrib": "Maghrib",
	"isha":    "Isha",
}
