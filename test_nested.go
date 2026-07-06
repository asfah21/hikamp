//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	scheduleInfoKey := "dailyScheduleInfo"
	scheduleInfo := map[string]interface{}{
		"startTime": "2026-07-06",
		"stopTime":  "2027-07-06",
		"dailyScheduleList": []map[string]interface{}{
			{
				"beginTime": "12:00:00+08:00",
				"endTime":   "12:05:00+08:00",
				"playMode":  "order",
				"operation": map[string]interface{}{
					"audioSource":   "customAudio",
					"customAudioID": []int{1},
					"audioLevel":    5,
					"audioVolume":   50,
				},
			},
		},
	}

	payload := map[string]interface{}{
		"broadcastPlanSchemeList": []map[string]interface{}{
			{
				"planSchemeID":   "sch_1_dan",
				"enabled":        true,
				"planSchemeName": "dan",
				"audioOutID":     []int{1},
				scheduleInfoKey:  scheduleInfo,
			},
		},
		"terminalInfoList": []map[string]interface{}{
			{
				"terminalID": 1,
				"audioOutID": []int{1},
			},
		},
	}

	jsonData, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(jsonData))
}
