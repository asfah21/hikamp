package hikvision

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DigestAuth handles HTTP Digest Authentication for Hikvision ISAPI
type DigestAuth struct {
	Username  string
	Password  string
	Realm     string
	Nonce     string
	Opaque    string
	QOP       string
	Algorithm string
}

// ParseDigestHeader parses the WWW-Authenticate header from a 401 response
func ParseDigestHeader(header string) *DigestAuth {
	da := &DigestAuth{}
	if !strings.HasPrefix(header, "Digest ") {
		return nil
	}

	header = strings.TrimPrefix(header, "Digest ")
	parts := strings.Split(header, ", ")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		switch key {
		case "realm":
			da.Realm = value
		case "nonce":
			da.Nonce = value
		case "opaque":
			da.Opaque = value
		case "qop":
			da.QOP = value
		case "algorithm":
			da.Algorithm = value
		}
	}
	return da
}

// GenerateAuthorizationHeader generates the Authorization header value
func (da *DigestAuth) GenerateAuthorizationHeader(method, uri string) string {
	if da.Algorithm == "" {
		da.Algorithm = "MD5"
	}

	// Generate cnonce
	cnonceBytes := make([]byte, 8)
	rand.Read(cnonceBytes)
	cnonce := hex.EncodeToString(cnonceBytes)

	// HA1 = MD5(username:realm:password)
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", da.Username, da.Realm, da.Password)))
	ha1Str := hex.EncodeToString(ha1[:])

	// HA2 = MD5(method:uri)
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))
	ha2Str := hex.EncodeToString(ha2[:])

	// Response = MD5(HA1:nonce:nonceCount:cnonce:qop:HA2)
	nc := "00000001"
	response := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1Str, da.Nonce, nc, cnonce, da.QOP, ha2Str)))
	responseStr := hex.EncodeToString(response[:])

	auth := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", algorithm=%s, response="%s", qop=%s, nc=%s, cnonce="%s"`,
		da.Username, da.Realm, da.Nonce, uri, da.Algorithm, responseStr, da.QOP, nc, cnonce)

	if da.Opaque != "" {
		auth += fmt.Sprintf(`, opaque="%s"`, da.Opaque)
	}

	return auth
}

// DoRequest performs an HTTP request with Digest Authentication.
// For POST requests, bodyBytes should be provided so the body can be re-read
// for the authenticated retry. If bodyBytes is nil, the body will be read from
// the body reader (but only once).
func DoRequest(client *http.Client, method, url string, username, password string, body io.Reader, contentType string) (*http.Response, error) {
	// Pre-read body bytes so we can reuse them for the retry
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}
	}

	// First attempt without auth to get the digest challenge
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	defer resp.Body.Close()

	// Parse WWW-Authenticate header
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		// No auth header, return the 401 response as-is
		return resp, nil
	}

	da := ParseDigestHeader(authHeader)
	if da == nil {
		return resp, nil
	}
	da.Username = username
	da.Password = password

	// Create new request with digest auth, reusing the pre-read body bytes
	req2, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated request: %w", err)
	}

	if contentType != "" {
		req2.Header.Set("Content-Type", contentType)
	}

	auth := da.GenerateAuthorizationHeader(method, url)
	req2.Header.Set("Authorization", auth)

	return client.Do(req2)
}
