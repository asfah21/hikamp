package hikvision

// Payload structs for Hikvision ISAPI AddPlanScheme endpoint.
// IMPORTANT: Hikvision Web UI uses NON-standard JSON field naming.
// Key differences from standard camelCase:
//   - "dailyScheduleInfo" (capital 'S') — NOT "dailyscheduleInfo"
//   - "planSchemeName" is always sent by Web UI
//   - startTime/stopTime format: "YYYY-MM-DD+08:00" (PLUS separator)
//   - beginTime/endTime format: "HH:MM:SS+08:00" (PLUS separator)
//   - "weklyScheduleInfo" — TYPO "wekly" (not "weekly") from official Web UI

// AddPlanSchemePayload is the top-level request payload
type AddPlanSchemePayload struct {
	BroadcastPlanSchemeList []BroadcastPlanScheme `json:"broadcastPlanSchemeList"`
	TerminalInfoList        []TerminalInfo        `json:"terminalInfoList"`
}

// BroadcastPlanScheme represents a single plan scheme entry
type BroadcastPlanScheme struct {
	PlanSchemeID       string              `json:"planSchemeID"`
	PlanSchemeName     string              `json:"planSchemeName,omitempty"`
	Enabled            bool                `json:"enabled"`
	AudioOutID         []int               `json:"audioOutID"`
	DailyScheduleInfo  *DailyScheduleInfo  `json:"dailyScheduleInfo,omitempty"`
	WeeklyScheduleInfo *WeeklyScheduleInfo `json:"weklyScheduleInfo,omitempty"`
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
	BeginTime   string    `json:"beginTime"`
	EndTime     string    `json:"endTime"`
	PlayNowTime string    `json:"playNowTime,omitempty"`
	PlayMode    string    `json:"playMode"`
	Operation   Operation `json:"operation"`
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
