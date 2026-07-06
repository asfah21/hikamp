package models

// AudioFile represents an uploaded audio file
type AudioFile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`     // Prayer, Attendance, Announcement, Emergency, Custom
	Duration    int    `json:"duration"`     // in seconds
	DurationStr string `json:"duration_str"` // formatted as mm:ss
	FileSize    int64  `json:"file_size"`
	SampleRate  int    `json:"sample_rate"` // in Hz
	FilePath    string `json:"file_path"`
	DeviceID    *int   `json:"device_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
