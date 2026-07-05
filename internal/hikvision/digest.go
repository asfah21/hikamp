package hikvision

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/icholy/digest"
)

// NewDigestClient creates an HTTP client that uses Digest Authentication.
// This is the standard HTTP Digest (RFC 7616) implementation used by Hikvision ISAPI.
func NewDigestClient(username, password string) *http.Client {
	transport := &digest.Transport{
		Username: username,
		Password: password,
		Transport: &http.Transport{
			MaxIdleConns:       5,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
}

// DoRequest performs an HTTP request with Digest Authentication.
// Uses the standard RFC 7616 digest implementation from github.com/icholy/digest.
func DoRequest(client *http.Client, method, url string, username, password string, body io.Reader, contentType string) (*http.Response, error) {
	// Create a digest-aware client for this request
	digestClient := NewDigestClient(username, password)
	// Copy the timeout from the original client
	digestClient.Timeout = client.Timeout

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := digestClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}
