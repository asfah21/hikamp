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
// Uses POST method with a payload matching the official Hikvision Web UI format.
// From Web UI JS: o.a.WSDK_SetDeviceConfig("SearchPlanScheme",null,{type:"POST",data:...})
func (c *Client) SearchPlanScheme() (interface{}, error) {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/SearchPlanScheme?format=json"

	// Payload matching Web UI format
	payload := map[string]interface{}{
		"terminalInfoList": []map[string]interface{}{
			{
				"terminalID": 1,
			},
		},
		"planSchemeID": []string{"1"},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search payload: %w", err)
	}

	resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
	if err != nil {
		return nil, fmt.Errorf("search plan scheme request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonData), resp, body)
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
				"planSchemeID": planSchemeID,
				"enabled":      false,
				"audioOutID":   []int{1},
				"dailyScheduleInfo": map[string]interface{}{
					"startTime":         "2000-01-01 00:00",
					"stopTime":          "2000-01-01 00:00",
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

// ModifyPlanScheme creates or updates a broadcast plan scheme on the Hikvision device.
// Uses ModifyPlanScheme endpoint which acts as an upsert — creates if not exists, updates if exists.
// This is safer than DeletePlanScheme + CreateSchedule because it doesn't remove unrelated schedules.
func (c *Client) ModifyPlanScheme(payload interface{}) error {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/ModifyPlanScheme?format=json"
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	log.Printf("[HIKVISION] ModifyPlanScheme payload: %s", string(jsonData))

	resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
	if err != nil {
		return fmt.Errorf("modify plan scheme request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonData), resp, body)
		return fmt.Errorf("modify plan scheme failed with status %d: %s", resp.StatusCode, string(body))
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

// HikvisionAudioInfo represents audio file info from Hikvision device
type HikvisionAudioInfo struct {
	CustomAudioID     int    `json:"customAudioID"`
	CustomAudioName   string `json:"customAudioName"`
	CustomAudioPath   string `json:"customAudioPath"`
	AudioFileFormat   string `json:"audioFileFormat"`
	AudioFileSize     int    `json:"audioFileSize"`
	AudioFileDuration int    `json:"audioFileDuration"`
	Duration          int    // computed from AudioFileDuration
	DurationStr       string // formatted as mm:ss
	HikvisionPath     string // alias for CustomAudioPath
}

// SearchAudio searches for audio files on the device
func (c *Client) SearchAudio() ([]HikvisionAudioInfo, error) {
	url := c.BaseURL + "/ISAPI/AccessControl/EventCardLinkageCfg/CustomAudio?format=json"
	resp, err := c.doRequestWithRetry("GET", url, nil, "", 1)
	if err != nil {
		return nil, fmt.Errorf("search audio request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		logFailedResponse("GET", url, "", resp, body)
		return nil, fmt.Errorf("search audio failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var result struct {
		CustomAudioInfoList []HikvisionAudioInfo `json:"CustomAudioInfoList"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search audio response: %w", err)
	}

	// Fill computed fields
	for i := range result.CustomAudioInfoList {
		info := &result.CustomAudioInfoList[i]
		info.Duration = info.AudioFileDuration
		info.HikvisionPath = info.CustomAudioPath
		if info.Duration > 0 {
			minutes := info.Duration / 60
			seconds := info.Duration % 60
			info.DurationStr = fmt.Sprintf("%d:%02d", minutes, seconds)
		}
	}

	return result.CustomAudioInfoList, nil
}

// SearchAudioByID searches for a specific audio file on the device by its customAudioID
func (c *Client) SearchAudioByID(audioID int) (*HikvisionAudioInfo, error) {
	audioList, err := c.SearchAudio()
	if err != nil {
		return nil, err
	}

	for _, info := range audioList {
		if info.CustomAudioID == audioID {
			return &info, nil
		}
	}

	return nil, fmt.Errorf("audio with ID %d not found on device", audioID)
}

// UploadAudio uploads an audio file to the device using multipart/form-data.
// Returns the customAudioID assigned by the device.
// Endpoint: POST /ISAPI/AccessControl/EventCardLinkageCfg/CustomAudio?format=json
// Format verified from official Hikvision Web UI Network request:
//   - Part 1: JSON field "CustomAudioInfo" with name, format, size
//   - Part 2: File field "audioData" with Content-Type: audio/mpeg
func (c *Client) UploadAudio(audioData io.Reader, filename string) (int, error) {
	url := c.BaseURL + "/ISAPI/AccessControl/EventCardLinkageCfg/CustomAudio?format=json"

	// Read the audio data to get its size
	audioBytes, err := io.ReadAll(audioData)
	if err != nil {
		return 0, fmt.Errorf("failed to read audio data: %w", err)
	}

	// Determine file format from extension
	fileFormat := "mp3"
	if strings.HasSuffix(strings.ToLower(filename), ".wav") {
		fileFormat = "wav"
	}

	// Build multipart form data matching Web UI format
	var requestBody bytes.Buffer
	writer := NewMultipartWriter(&requestBody)

	// Part 1: JSON metadata
	jsonPart := map[string]interface{}{
		"CustomAudioInfo": map[string]interface{}{
			"customAudioName": filename,
			"audioFileFormat": fileFormat,
			"audioFileSize":   len(audioBytes),
		},
	}
	jsonBytes, _ := json.Marshal(jsonPart)
	if err := writer.WriteField("CustomAudioInfo", string(jsonBytes)); err != nil {
		return 0, fmt.Errorf("failed to write JSON field: %w", err)
	}

	// Part 2: Audio file data
	if err := writer.WriteFile("audioData", filename, "audio/mpeg", audioBytes); err != nil {
		return 0, fmt.Errorf("failed to write audio file: %w", err)
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	log.Printf("[HIKVISION] Uploading audio: %s (%d bytes, format: %s)", filename, len(audioBytes), fileFormat)

	resp, err := DoRequest(c.DigestClient, "POST", url, &requestBody, contentType)
	if err != nil {
		return 0, fmt.Errorf("upload audio request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonBytes), resp, body)
		return 0, fmt.Errorf("upload audio failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get the assigned audio ID
	var uploadResult struct {
		CustomAudioInfo struct {
			CustomAudioID int `json:"customAudioID"`
		} `json:"CustomAudioInfo"`
	}
	if err := json.Unmarshal(body, &uploadResult); err == nil && uploadResult.CustomAudioInfo.CustomAudioID > 0 {
		log.Printf("[HIKVISION] Audio uploaded successfully: %s (ID: %d)", filename, uploadResult.CustomAudioInfo.CustomAudioID)
		return uploadResult.CustomAudioInfo.CustomAudioID, nil
	}

	// If we can't parse the response, search for the audio by name to get its ID
	log.Printf("[HIKVISION] Audio uploaded successfully: %s (searching for assigned ID)", filename)
	audioList, err := c.SearchAudio()
	if err != nil {
		return 0, fmt.Errorf("audio uploaded but failed to get ID: %w", err)
	}

	for _, info := range audioList {
		if info.CustomAudioName == filename {
			return info.CustomAudioID, nil
		}
	}

	return 0, fmt.Errorf("audio uploaded but could not determine assigned ID")
}

// DeleteAudio deletes an audio file from the device by its customAudioID.
// Uses the same endpoint as upload but with a DELETE-style payload.
// From Web UI JS: WSDK_SetDeviceConfig("deleteCustomAudio", null, {type:"POST", data:{customAudioIDList:[id]}})
// Endpoint: POST /ISAPI/AccessControl/EventCardLinkageCfg/CustomAudio?format=json
func (c *Client) DeleteAudio(audioID int) error {
	url := c.BaseURL + "/ISAPI/AccessControl/EventCardLinkageCfg/CustomAudio?format=json"

	payload := map[string]interface{}{
		"customAudioIDList": []int{audioID},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delete audio payload: %w", err)
	}

	resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
	if err != nil {
		return fmt.Errorf("delete audio request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonData), resp, body)
		return fmt.Errorf("delete audio failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[HIKVISION] Audio ID %d deleted successfully", audioID)
	return nil
}

// DeleteAudioBatch deletes multiple audio files from the device in a single request.
// Uses the same endpoint as DeleteAudio but with a list of IDs.
func (c *Client) DeleteAudioBatch(audioIDs []int) error {
	if len(audioIDs) == 0 {
		return nil
	}

	url := c.BaseURL + "/ISAPI/AccessControl/EventCardLinkageCfg/CustomAudio?format=json"

	payload := map[string]interface{}{
		"customAudioIDList": audioIDs,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal batch delete audio payload: %w", err)
	}

	resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
	if err != nil {
		return fmt.Errorf("batch delete audio request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonData), resp, body)
		return fmt.Errorf("batch delete audio failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[HIKVISION] %d audio(s) deleted successfully", len(audioIDs))
	return nil
}

// getLocationFromOffset converts a timezone offset string like "07:00" or "-05:00"
// to a *time.Location. This is used to convert server time to the target timezone
// before formatting timestamps for the Hikvision device.
func getLocationFromOffset(offset string) *time.Location {
	// Parse "+HH:MM" or "-HH:MM" format
	hours := 0
	mins := 0
	if len(offset) >= 5 {
		sign := 1
		start := 0
		if offset[0] == '-' {
			sign = -1
			start = 1
		} else if offset[0] == '+' {
			start = 1
		}
		hours, _ = fmt.Sscanf(offset[start:start+2], "%d", &hours)
		fmt.Sscanf(offset[start+3:start+5], "%d", &mins)
		hours *= sign
		mins *= sign
	}
	totalSecs := hours*3600 + mins*60
	return time.FixedZone(fmt.Sprintf("UTC%+03d:%02d", hours, mins), totalSecs)
}

// BroadcastNow broadcasts audio immediately using ModifyPlanScheme with an immediate schedule.
// Creates a temporary schedule that starts in ~62 minutes and ends after the specified duration.
// Uses ModifyPlanScheme (non-destructive) — adds the broadcast without touching existing schedules.
// startTime/stopTime are set to today only (1-day schedule).
func (c *Client) BroadcastNow(audioID int, volume int, durationMinutes int) error {
	return c.BroadcastNowWithTimezone(audioID, volume, durationMinutes, "00:00")
}

// BroadcastNowWithTimezone broadcasts audio immediately with a configurable timezone offset.
// The timezoneOffset should be in format like "07:00" or "-05:00" (WITHOUT "+" prefix).
//
// Uses a dual-strategy approach:
//  1. Strategy 1: Try ModifyPlanScheme first (non-destructive, preserves other schedules)
//  2. Strategy 2: If ModifyPlanScheme fails (403 on some firmware), fallback to:
//     SearchPlanScheme → merge existing schedules + broadcast_now → AddPlanScheme
//     This preserves existing schedules while adding the broadcast_now entry.
//
// Uses map[string]interface{} payload matching the official Hikvision Web UI format.
func (c *Client) BroadcastNowWithTimezone(audioID int, volume int, durationMinutes int, timezoneOffset string) error {
	now := time.Now()

	// Add 62 minutes to beginTime to compensate for timezone offset differences
	// between the server and the Hikvision device.
	beginTime := now.Add(62*time.Minute).Format("15:04:05") + "+" + timezoneOffset
	endTime := now.Add(62*time.Minute+time.Duration(durationMinutes)*time.Minute).Format("15:04:05") + "+" + timezoneOffset

	// Web UI uses "YYYY-MM-DD+HH:MM" format for startTime/stopTime
	// where HH:MM is the TIMEZONE OFFSET (e.g., "08:00"), NOT the current time
	// Both start and stop are set to today only — 1-day schedule.
	today := now.Format("2006-01-02") + "+" + timezoneOffset

	planSchemeID := fmt.Sprintf("broadcast_now_%d", now.Unix())

	broadcastNowScheme := map[string]interface{}{
		"planSchemeID":   planSchemeID,
		"planSchemeName": "Broadcast Now",
		"enabled":        true,
		"audioOutID":     []int{1},
		"dailyScheduleInfo": map[string]interface{}{
			"startTime": today,
			"stopTime":  today,
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
	}

	// Strategy 1: Try ModifyPlanScheme first (non-destructive)
	payload := map[string]interface{}{
		"broadcastPlanSchemeList": []map[string]interface{}{
			broadcastNowScheme,
		},
		"terminalInfoList": []map[string]interface{}{
			{
				"terminalID": 1,
				"audioOutID": []int{1},
			},
		},
	}

	err := c.ModifyPlanScheme(payload)
	if err == nil {
		return nil
	}

	// If ModifyPlanScheme fails (403 on some firmware), log and fall back to Strategy 2
	log.Printf("[HIKVISION] ModifyPlanScheme failed for broadcast_now: %v — falling back to SearchPlanScheme + AddPlanScheme", err)

	// Strategy 2: SearchPlanScheme → merge existing + broadcast_now → AddPlanScheme
	// This preserves existing schedules while adding the broadcast_now entry.
	return c.broadcastNowWithAddPlanScheme(broadcastNowScheme, planSchemeID)
}

// broadcastNowWithAddPlanScheme is the fallback strategy for BroadcastNow.
// It searches existing schedules on the device, merges them with the broadcast_now
// schedule, and sends everything via AddPlanScheme (which replaces all schedules).
// This preserves existing schedules while adding the broadcast_now entry.
func (c *Client) broadcastNowWithAddPlanScheme(broadcastNowScheme map[string]interface{}, planSchemeID string) error {
	// Step 1: Search all existing schedules on the device
	existingSchemes := []map[string]interface{}{}
	schemes, searchErr := c.SearchPlanScheme()
	if searchErr == nil {
		if schemesMap, ok := schemes.(map[string]interface{}); ok {
			if list, ok := schemesMap["broadcastPlanSchemeList"].([]interface{}); ok {
				for _, item := range list {
					if scheme, ok := item.(map[string]interface{}); ok {
						existingSchemes = append(existingSchemes, scheme)
					}
				}
			}
		}
	} else {
		log.Printf("[HIKVISION] SearchPlanScheme failed in fallback: %v — proceeding with broadcast_now only", searchErr)
	}

	// Step 2: Remove any existing broadcast_now scheme with the same ID (to avoid duplicates)
	filteredSchemes := []map[string]interface{}{}
	for _, scheme := range existingSchemes {
		if id, ok := scheme["planSchemeID"].(string); ok && id == planSchemeID {
			continue // skip old broadcast_now with same ID
		}
		filteredSchemes = append(filteredSchemes, scheme)
	}

	// Step 3: Add the broadcast_now scheme to the list
	allSchemes := append(filteredSchemes, broadcastNowScheme)

	// Step 4: Send everything via AddPlanScheme
	addPayload := map[string]interface{}{
		"broadcastPlanSchemeList": allSchemes,
		"terminalInfoList": []map[string]interface{}{
			{
				"terminalID": 1,
				"audioOutID": []int{1},
			},
		},
	}

	return c.CreateSchedule(addPayload)
}

// StopBroadcast stops all active broadcasts on the device.
// Uses SearchPlanScheme to find all active schedules, then disables them
// via ModifyPlanScheme with enabled=false.
// This stops audio output without deleting the schedules.
func (c *Client) StopBroadcast() error {
	// Search for all existing plan schemes
	schemes, err := c.SearchPlanScheme()
	if err != nil {
		return fmt.Errorf("failed to search plan schemes: %w", err)
	}

	// Parse the response to extract planSchemeIDs
	schemesMap, ok := schemes.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format from SearchPlanScheme")
	}

	list, ok := schemesMap["broadcastPlanSchemeList"].([]interface{})
	if !ok || len(list) == 0 {
		// No schedules found, nothing to stop
		return nil
	}

	var lastErr error
	for _, item := range list {
		scheme, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		planSchemeID, ok := scheme["planSchemeID"].(string)
		if !ok || planSchemeID == "" {
			continue
		}

		// Disable this schedule via ModifyPlanScheme
		err := c.disablePlanScheme(planSchemeID)
		if err != nil {
			log.Printf("[HIKVISION] Failed to disable schedule '%s': %v", planSchemeID, err)
			lastErr = err
			continue
		}
		log.Printf("[HIKVISION] Disabled schedule '%s'", planSchemeID)
	}

	return lastErr
}

// disablePlanScheme disables a single plan scheme by ID.
// Uses ModifyPlanScheme with enabled=false to stop it without deleting.
func (c *Client) disablePlanScheme(planSchemeID string) error {
	url := c.BaseURL + "/ISAPI/VideoIntercom/broadcast/ModifyPlanScheme?format=json"

	payload := map[string]interface{}{
		"broadcastPlanSchemeList": []map[string]interface{}{
			{
				"planSchemeID": planSchemeID,
				"enabled":      false,
				"audioOutID":   []int{1},
				"dailyScheduleInfo": map[string]interface{}{
					"startTime":         "2000-01-01 00:00",
					"stopTime":          "2000-01-01 00:00",
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
		return fmt.Errorf("failed to marshal disable payload: %w", err)
	}

	resp, err := c.doRequestWithRetry("POST", url, bytes.NewReader(jsonData), "application/json", 1)
	if err != nil {
		return fmt.Errorf("disable plan scheme request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logFailedResponse("POST", url, string(jsonData), resp, body)
		return fmt.Errorf("disable plan scheme failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
