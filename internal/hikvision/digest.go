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
// Creates a fresh digest client for each request to avoid body-read issues
// that can occur when the digest transport retries after a 401 challenge.
func DoRequest(client *http.Client, method, url string, body io.Reader, contentType string) (*http.Response, error) {
	// Read body into bytes so we can create multiple readers
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}
	}

	// Create a fresh digest client for this request.
	// This avoids body-read issues when the digest transport retries after 401.
	// We extract credentials from the provided client's transport.
	username, password := extractCredentials(client)
	freshClient := NewDigestClient(username, password)
	freshClient.Timeout = client.Timeout

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set GetBody so the digest transport can re-read the body on 401 retry.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := freshClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// extractCredentials extracts username and password from a digest-aware HTTP client.
func extractCredentials(client *http.Client) (string, string) {
	if transport, ok := client.Transport.(*digest.Transport); ok {
		return transport.Username, transport.Password
	}
	return "", ""
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
