package hikvision

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client is the reusable Hikvision ISAPI client.
// Uses a single Digest-aware HTTP client for connection reuse and nonce caching.
type Client struct {
	DigestClient *http.Client
	BaseURL      string
	Username     string
	Password     string
}

// NewClient creates a new Hikvision client with a Digest-aware HTTP client.
// The transport is reused across requests to enable:
// - Connection pooling (TCP connection reuse)
// - Digest nonce caching (avoids repeated 401 handshakes)
func NewClient(ip string, port int, username, password string) *Client {
	baseURL := fmt.Sprintf("http://%s:%d", ip, port)
	return &Client{
		DigestClient: NewDigestClient(username, password),
		BaseURL:      baseURL,
		Username:     username,
		Password:     password,
	}
}

// doRequestWithRetry performs an HTTP request with retry logic for retryable errors.
// Retries only for: connection errors, timeouts, 5xx server errors.
// Does NOT retry for: 4xx client errors (including 401, 404, 400).
func (c *Client) doRequestWithRetry(method, url string, body io.Reader, contentType string, maxRetries int) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// For retries, we need to re-create the body reader since it may have been consumed
		var bodyReader io.Reader
		if body != nil {
			// Try to use a fresh reader for retries
			if seeker, ok := body.(io.Seeker); ok {
				seeker.Seek(0, io.SeekStart)
				bodyReader = body
			} else {
				// If body can't be reset, we can only retry if it's the first attempt
				if attempt > 0 {
					return nil, fmt.Errorf("cannot retry: request body is not seekable (last error: %w)", lastErr)
				}
				bodyReader = body
			}
		}

		resp, lastErr = DoRequest(c.DigestClient, method, url, bodyReader, contentType)
		if lastErr == nil {
			// Request succeeded at HTTP level, check status code
			if resp.StatusCode < 500 {
				// Not a server error, return as-is (even 4xx)
				return resp, nil
			}

			// Server error (5xx) — read body for logging, then retry
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			logFailedResponse(method, url, "", resp, respBody)
			lastErr = fmt.Errorf("server error: status %d, body: %s", resp.StatusCode, string(respBody))
		} else {
			// Connection-level error (timeout, DNS, etc.)
			log.Printf("[HIKVISION RETRY] Attempt %d/%d failed: %v", attempt+1, maxRetries+1, lastErr)
		}

		if attempt < maxRetries {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Printf("[HIKVISION RETRY] Waiting %v before retry %d/%d", backoff, attempt+2, maxRetries+1)
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// DeviceInfo reads device information from Hikvision device
func (c *Client) DeviceInfo() (map[string]string, error) {
	url := c.BaseURL + "/ISAPI/System/deviceInfo"
	resp, err := c.doRequestWithRetry("GET", url, nil, "", 1)
	if err != nil {
		return nil, fmt.Errorf("device info request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		logFailedResponse("GET", url, "", resp, body)
		return nil, fmt.Errorf("device info failed with status %d: %s", resp.StatusCode, string(body))
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
	resp, err := c.doRequestWithRetry("GET", url, nil, "", 1)
	if err != nil {
		return nil, fmt.Errorf("search plan scheme request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		logFailedResponse("GET", url, "", resp, body)
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

	resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
	if err != nil {
		return fmt.Errorf("delete plan scheme request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonData), resp, body)
		return fmt.Errorf("delete plan scheme failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateSchedule creates a broadcast schedule on the Hikvision device.
// Uses AddPlanScheme endpoint with the provided payload.
func (c *Client) CreateSchedule(payload interface{}) error {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/AddPlanScheme?format=json"
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	log.Printf("[HIKVISION] CreateSchedule payload: %s", string(jsonData))

	resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
	if err != nil {
		return fmt.Errorf("create schedule request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonData), resp, body)
		return fmt.Errorf("create schedule failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateScheduleWithRetry creates a broadcast schedule with retry logic for device-busy scenarios.
// Retries up to maxRetries times with exponential backoff for 4xx/5xx errors.
func (c *Client) CreateScheduleWithRetry(payload interface{}, maxRetries int) error {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/AddPlanScheme?format=json"
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	log.Printf("[HIKVISION] CreateSchedule payload: %s", string(jsonData))

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
		if err != nil {
			lastErr = fmt.Errorf("create schedule request failed: %w", err)
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode < 400 {
				return nil // success
			}

			logFailedResponse("POST", url, string(jsonData), resp, body)
			lastErr = fmt.Errorf("create schedule failed with status %d: %s", resp.StatusCode, string(body))

			// Don't retry if it's a JSON content error (structural issue won't be fixed by retry)
			if resp.StatusCode == 400 && strings.Contains(string(body), "badJsonContent") {
				return lastErr
			}
		}

		if attempt < maxRetries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Printf("[HIKVISION RETRY] Waiting %v before retry %d/%d", backoff, attempt+2, maxRetries+1)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("create schedule failed after %d retries: %w", maxRetries, lastErr)
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
// Uses map[string]interface{} payload (proven to work with Hikvision firmware).
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
