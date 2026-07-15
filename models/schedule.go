package models

// ScheduleEntry represents a single time range + audio + volume in a schedule
type ScheduleEntry struct {
	ID         int    `json:"id"`
	ScheduleID int    `json:"schedule_id"`
	AudioID    int    `json:"audio_id"`
	BeginTime  string `json:"begin_time"`
	EndTime    string `json:"end_time"`
	Volume     int    `json:"volume"`
}

// ScheduleDevice represents a device target for a schedule
type ScheduleDevice struct {
	ID         int `json:"id"`
	ScheduleID int `json:"schedule_id"`
	DeviceID   int `json:"device_id"`
}

// BroadcastSchedule represents a scheduled broadcast with multiple entries and devices
type BroadcastSchedule struct {
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	ScheduleType string           `json:"schedule_type"` // daily, weekly, specific_date
	Enabled      bool             `json:"enabled"`
	StartDate    *string          `json:"start_date"`    // optional date range start
	EndDate      *string          `json:"end_date"`      // optional date range end
	DayOfWeek    *int             `json:"day_of_week"`   // 1=Monday ... 7=Sunday (for weekly)
	SpecificDate *string          `json:"specific_date"` // for specific_date type
	Entries      []ScheduleEntry  `json:"entries"`
	Devices      []ScheduleDevice `json:"devices"`
	CreatedAt    string           `json:"created_at"`
	UpdatedAt    string           `json:"updated_at"`
}
