package hikvision

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the reusable Hikvision ISAPI client
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	Username   string
	Password   string
}

// NewClient creates a new Hikvision client
func NewClient(ip string, port int, username, password string) *Client {
	baseURL := fmt.Sprintf("http://%s:%d", ip, port)
	return &Client{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		BaseURL:  baseURL,
		Username: username,
		Password: password,
	}
}

// DeviceInfo reads device information from Hikvision device
func (c *Client) DeviceInfo() (map[string]string, error) {
	url := c.BaseURL + "/ISAPI/System/deviceInfo"
	resp, err := DoRequest(c.HTTPClient, "GET", url, c.Username, c.Password, nil, "")
	if err != nil {
		return nil, fmt.Errorf("device info request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse XML response - use a generic decoder to handle namespaces
	// Hikvision devices often include xmlns attributes that break strict XML matching
	result := map[string]string{}
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var currentElement string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse device info XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			currentElement = t.Name.Local
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" && currentElement != "" {
				switch currentElement {
				case "deviceName":
					result["deviceName"] = text
				case "deviceID":
					result["deviceID"] = text
				case "deviceType":
					result["deviceType"] = text
				case "serialNumber":
					result["serialNumber"] = text
				case "firmwareVersion":
					result["firmwareVersion"] = text
				case "firmwareReleasedDate":
					result["firmwareReleasedDate"] = text
				}
			}
		}
	}

	return result, nil
}

// TestConnection tests connection to the device
func (c *Client) TestConnection() error {
	_, err := c.DeviceInfo()
	return err
}

// SearchSchedule searches for broadcast schedules
func (c *Client) SearchSchedule() (interface{}, error) {
	// Endpoint not verified - see docs/hikvision.md
	// Placeholder for future implementation
	return nil, fmt.Errorf("SearchSchedule not yet implemented - inspect Hikvision Web UI Network tab first")
}

// CreateSchedule creates a broadcast schedule
func (c *Client) CreateSchedule(payload interface{}) error {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/AddPlanScheme?format=json"
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := DoRequest(c.HTTPClient, "POST", url, c.Username, c.Password, bytes.NewReader(jsonData), "application/json")
	if err != nil {
		return fmt.Errorf("create schedule request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("create schedule failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateSchedule updates a broadcast schedule
func (c *Client) UpdateSchedule(payload interface{}) error {
	// Endpoint not verified - see docs/hikvision.md
	// Placeholder for future implementation
	return fmt.Errorf("UpdateSchedule not yet implemented - inspect Hikvision Web UI Network tab first")
}

// DeleteSchedule deletes a broadcast schedule
func (c *Client) DeleteSchedule(scheduleID string) error {
	// Endpoint not verified - see docs/hikvision.md
	// Placeholder for future implementation
	return fmt.Errorf("DeleteSchedule not yet implemented - inspect Hikvision Web UI Network tab first")
}

// SearchAudio searches for audio files on the device
func (c *Client) SearchAudio() (interface{}, error) {
	// Endpoint not verified - see docs/hikvision.md
	// Placeholder for future implementation
	return nil, fmt.Errorf("SearchAudio not yet implemented - inspect Hikvision Web UI Network tab first")
}

// UploadAudio uploads an audio file to the device
func (c *Client) UploadAudio(audioData io.Reader, filename string) error {
	// Endpoint not verified - see docs/hikvision.md
	// Placeholder for future implementation
	return fmt.Errorf("UploadAudio not yet implemented - inspect Hikvision Web UI Network tab first")
}

// DeleteAudio deletes an audio file from the device
func (c *Client) DeleteAudio(audioID int) error {
	// Endpoint not verified - see docs/hikvision.md
	// Placeholder for future implementation
	return fmt.Errorf("DeleteAudio not yet implemented - inspect Hikvision Web UI Network tab first")
}

// BroadcastNow broadcasts audio immediately using AddPlanScheme with an immediate schedule.
// Creates a temporary schedule that starts now and ends after the specified duration.
// Uses the verified payload structure from the official Hikvision Web UI.
func (c *Client) BroadcastNow(audioID int, volume int, durationMinutes int) error {
	now := time.Now()
	// Use +08:00 timezone format as seen in the verified Web UI example
	loc := time.FixedZone("UTC+8", 8*60*60)
	nowLocal := now.In(loc)
	beginTime := nowLocal.Format("15:04:05-07:00")
	endTime := nowLocal.Add(time.Duration(durationMinutes) * time.Minute).Format("15:04:05-07:00")
	dateStr := nowLocal.Format("2006-01-02")

	payload := map[string]interface{}{
		"broadcastPlanSchemeList": []map[string]interface{}{
			{
				"planSchemeID":   fmt.Sprintf("broadcast_now_%d", now.Unix()),
				"enabled":        true,
				"planSchemeName": "Broadcast Now",
				"audioOutID":     []int{1},
				"dailyScheduleInfo": map[string]interface{}{
					"startTime": dateStr,
					"stopTime":  dateStr,
					"dailyScheduleList": []map[string]interface{}{
						{
							"beginTime": beginTime,
							"endTime":   endTime,
							"playMode":  "order",
							"operation": map[string]interface{}{
								"audioSource":   "customAudio",
								"customAudioID": []int{audioID},
								"audioLevel":    5,
								"audioVolume":   volume,
							},
						},
					},
				},
			},
		},
		"terminalInfoList": []map[string]interface{}{
			{
				"terminalID": 1,
				"audioOutID": []int{1},
			},
		},
	}

	return c.CreateSchedule(payload)
}

// StopBroadcast stops the current broadcast
func (c *Client) StopBroadcast() error {
	// Endpoint not verified - see docs/hikvision.md
	// Placeholder for future implementation
	return fmt.Errorf("StopBroadcast not yet implemented - inspect Hikvision Web UI Network tab first")
}
