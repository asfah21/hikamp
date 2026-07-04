package models

// BroadcastSchedule represents a scheduled broadcast
type BroadcastSchedule struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	AudioID      int     `json:"audio_id"`
	DeviceID     int     `json:"device_id"`
	ScheduleType string  `json:"schedule_type"` // daily, weekly, specific_date
	BeginTime    string  `json:"begin_time"`
	EndTime      string  `json:"end_time"`
	Volume       int     `json:"volume"`
	Enabled      bool    `json:"enabled"`
	DayOfWeek    *int    `json:"day_of_week"`   // 1=Monday ... 7=Sunday (for weekly)
	SpecificDate *string `json:"specific_date"` // for specific_date type
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}
