package hikvision

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/icholy/digest"
)

// NewDigestClient creates an HTTP client that uses Digest Authentication.
// This is the standard HTTP Digest (RFC 7616) implementation used by Hikvision ISAPI.
// The transport is reused across requests to enable connection pooling and digest nonce caching.
func NewDigestClient(username, password string) *http.Client {
	transport := &digest.Transport{
		Username: username,
		Password: password,
		Transport: &http.Transport{
			MaxIdleConns:        5,
			MaxIdleConnsPerHost: 3,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		// Prevent following redirects automatically.
		// Hikvision devices may redirect POST requests, causing body loss.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// DoRequest performs an HTTP request with Digest Authentication.
// Uses the standard RFC 7616 digest implementation from github.com/icholy/digest.
// The provided client should be a digest-aware client (created via NewDigestClient)
// to ensure connection reuse and digest nonce caching.
//
// IMPORTANT: This function sets req.GetBody so that the digest transport can
// re-read the request body when retrying after a 401 (digest challenge).
// Without GetBody, the digest library would send an empty body on retry,
// causing "Invalid JSON Content" / "badJsonContent" errors from Hikvision devices.
func DoRequest(client *http.Client, method, url string, body io.Reader, contentType string) (*http.Response, error) {
	// Read body into bytes so we can create multiple readers (for GetBody)
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set GetBody so the digest transport can re-read the body on 401 retry.
	// This is critical for Hikvision devices that use Digest Authentication.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// logFailedResponse logs the full details of a failed response for debugging.
func logFailedResponse(method, url string, payload string, resp *http.Response, body []byte) {
	log.Printf("[HIKVISION FAIL] %s %s", method, url)
	if payload != "" {
		log.Printf("[HIKVISION FAIL] Payload: %s", payload)
	}
	log.Printf("[HIKVISION FAIL] HTTP Status: %d %s", resp.StatusCode, resp.Status)
	log.Printf("[HIKVISION FAIL] Response Body: %s", string(body))
}
