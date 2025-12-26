package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// LoginRequest is the request body for login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the response from login.
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Login authenticates with hub-core and returns a token.
func (c *Client) Login(username, password string) (*LoginResponse, error) {
	reqBody, err := json.Marshal(LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.post("/auth/login", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    "invalid username or password",
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &loginResp, nil
}

// TokenExpiry extracts the expiry time from a JWT token.
// Returns zero time if the token is invalid or has no expiry.
func TokenExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return time.Time{}
	}

	if claims.Exp == 0 {
		return time.Time{}
	}

	return time.Unix(claims.Exp, 0)
}

// IsTokenExpired checks if a token is expired.
// Returns true if the token is invalid or expired.
func IsTokenExpired(token string) bool {
	if token == "" {
		return true
	}

	expiry := TokenExpiry(token)
	if expiry.IsZero() {
		// Can't determine expiry, assume valid
		return false
	}

	// Add a small buffer (30 seconds) to avoid edge cases
	return time.Now().After(expiry.Add(-30 * time.Second))
}
