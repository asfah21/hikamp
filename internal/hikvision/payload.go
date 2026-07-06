package hikvision

// Payload structs for Hikvision ISAPI AddPlanScheme endpoint.
// Using structs instead of map[string]interface{} to control JSON key ordering,
// as some Hikvision firmware versions may be sensitive to key order.

// AddPlanSchemePayload is the top-level request payload
type AddPlanSchemePayload struct {
	BroadcastPlanSchemeList []BroadcastPlanScheme `json:"broadcastPlanSchemeList"`
	TerminalInfoList        []TerminalInfo        `json:"terminalInfoList"`
}

// BroadcastPlanScheme represents a single plan scheme entry
type BroadcastPlanScheme struct {
	PlanSchemeID       string              `json:"planSchemeID"`
	Enabled            bool                `json:"enabled"`
	PlanSchemeName     string              `json:"planSchemeName"`
	DailyScheduleInfo  *DailyScheduleInfo  `json:"dailyScheduleInfo,omitempty"`
	WeeklyScheduleInfo *WeeklyScheduleInfo `json:"weeklyScheduleInfo,omitempty"`
	AudioOutID         []int               `json:"audioOutID"`
}

// DailyScheduleInfo contains daily schedule configuration
type DailyScheduleInfo struct {
	StartTime         string          `json:"startTime"`
	StopTime          string          `json:"stopTime"`
	DailyScheduleList []ScheduleEntry `json:"dailyScheduleList"`
}

// WeeklyScheduleInfo contains weekly schedule configuration
type WeeklyScheduleInfo struct {
	StartTime          string              `json:"startTime"`
	StopTime           string              `json:"stopTime"`
	WeeklyScheduleList []WeeklyScheduleDay `json:"weeklyScheduleList"`
}

// WeeklyScheduleDay represents a day in weekly schedule
type WeeklyScheduleDay struct {
	DayOfWeek    int             `json:"dayOfWeek"`
	ScheduleList []ScheduleEntry `json:"scheduleList"`
}

// ScheduleEntry represents a single schedule time entry
type ScheduleEntry struct {
	BeginTime string    `json:"beginTime"`
	EndTime   string    `json:"endTime"`
	PlayMode  string    `json:"playMode"`
	Operation Operation `json:"operation"`
}

// Operation defines the audio playback operation
type Operation struct {
	AudioSource   string `json:"audioSource"`
	CustomAudioID []int  `json:"customAudioID"`
	AudioLevel    int    `json:"audioLevel"`
	AudioVolume   int    `json:"audioVolume"`
}

// TerminalInfo represents a terminal configuration
type TerminalInfo struct {
	TerminalID int   `json:"terminalID"`
	AudioOutID []int `json:"audioOutID"`
}
