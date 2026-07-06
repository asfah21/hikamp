package hikvision

// Payload structs for Hikvision ISAPI AddPlanScheme endpoint.
// IMPORTANT: Hikvision Web UI uses NON-standard JSON field naming.
// Key differences from standard camelCase:
//   - "dailyscheduleInfo" (lowercase 's') — NOT "dailyScheduleInfo"
//   - No "planSchemeName" field
//   - startTime/stopTime format: "YYYY-MM-DD HH:MM" (with time component)
//   - beginTime/endTime format: "HH:MM:SS HH:MM" (space separator, NOT "+")

// AddPlanSchemePayload is the top-level request payload
type AddPlanSchemePayload struct {
	BroadcastPlanSchemeList []BroadcastPlanScheme `json:"broadcastPlanSchemeList"`
	TerminalInfoList        []TerminalInfo        `json:"terminalInfoList"`
}

// BroadcastPlanScheme represents a single plan scheme entry
// Note: Web UI does NOT send "planSchemeName" field
type BroadcastPlanScheme struct {
	PlanSchemeID       string              `json:"planSchemeID"`
	Enabled            bool                `json:"enabled"`
	AudioOutID         []int               `json:"audioOutID"`
	DailyScheduleInfo  *DailyScheduleInfo  `json:"dailyscheduleInfo,omitempty"`
	WeeklyScheduleInfo *WeeklyScheduleInfo `json:"weeklyScheduleInfo,omitempty"`
}

// DailyScheduleInfo contains daily schedule configuration
// Note: JSON tag uses "dailyscheduleInfo" (lowercase 's') to match Web UI
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
