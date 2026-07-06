package models

// AudioFile represents an audio file synced from Hikvision device
type AudioFile struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	Category         string  `json:"category"`     // Prayer, Attendance, Announcement, Emergency, Custom
	Duration         int     `json:"duration"`     // in seconds (from Hikvision)
	DurationStr      string  `json:"duration_str"` // formatted as mm:ss
	FileSize         int64   `json:"file_size"`
	HikvisionAudioID *int    `json:"hikvision_audio_id"` // customAudioID from Hikvision device (nullable)
	HikvisionPath    *string `json:"hikvision_path"`     // path on Hikvision device (nullable)
	DeviceID         *int    `json:"device_id"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}
