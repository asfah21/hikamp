package models

// BroadcastLog records every broadcast event
type BroadcastLog struct {
	ID           int    `json:"id"`
	Time         string `json:"time"`
	DeviceID     int    `json:"device_id"`
	DeviceName   string `json:"device_name"`
	AudioID      int    `json:"audio_id"`
	AudioName    string `json:"audio_name"`
	Result       string `json:"result"` // success, failed
	Duration     int    `json:"duration"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	CreatedAt    string `json:"created_at"`
}
