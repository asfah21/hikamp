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

// SearchPlanScheme searches for existing broadcast plan schemes on the device.
// Uses the same endpoint pattern as AddPlanScheme but with GET method.
func (c *Client) SearchPlanScheme() (interface{}, error) {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/SearchPlanScheme?format=json"
	resp, err := DoRequest(c.HTTPClient, "GET", url, c.Username, c.Password, nil, "")
	if err != nil {
		return nil, fmt.Errorf("search plan scheme request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("search plan scheme failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse as JSON
	var result interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		return result, nil
	}

	return string(body), nil
}

// DeletePlanScheme deletes a broadcast plan scheme by its ID.
// Uses the ModifyPlanScheme endpoint with an empty enabled=false payload to remove it.
func (c *Client) DeletePlanScheme(planSchemeID string) error {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/ModifyPlanScheme?format=json"

	// Delete payload: set enabled=false and empty schedule to effectively remove it
	payload := map[string]interface{}{
		"broadcastPlanSchemeList": []map[string]interface{}{
			{
				"planSchemeID":   planSchemeID,
				"enabled":        false,
				"planSchemeName": planSchemeID,
				"audioOutID":     []int{1},
				"dailyScheduleInfo": map[string]interface{}{
					"startTime":         "2000-01-01",
					"stopTime":          "2000-01-01",
					"dailyScheduleList": []map[string]interface{}{},
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

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delete payload: %w", err)
	}

	resp, err := DoRequest(c.HTTPClient, "POST", url, c.Username, c.Password, bytes.NewReader(jsonData), "application/json")
	if err != nil {
		return fmt.Errorf("delete plan scheme request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete plan scheme failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
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
// Uses default +08:00 timezone.
func (c *Client) BroadcastNow(audioID int, volume int, durationMinutes int) error {
	return c.BroadcastNowWithTimezone(audioID, volume, durationMinutes, "+08:00")
}

// BroadcastNowWithTimezone broadcasts audio immediately with a configurable timezone offset.
// The timezoneOffset should be in format like "+07:00" or "+08:00".
func (c *Client) BroadcastNowWithTimezone(audioID int, volume int, durationMinutes int, timezoneOffset string) error {
	now := time.Now()

	beginTime := now.Format("15:04:05") + timezoneOffset
	endTime := now.Add(time.Duration(durationMinutes)*time.Minute).Format("15:04:05") + timezoneOffset
	dateStr := now.Format("2006-01-02")

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
