package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the HTTP client for hub-core API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates a new hub-core client.
func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetToken sets the auth token for requests.
func (c *Client) SetToken(token string) {
	c.token = token
}

// Token returns the current auth token.
func (c *Client) Token() string {
	return c.token
}

// BaseURL returns the base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// do executes an HTTP request with auth header injection.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

// get performs a GET request.
func (c *Client) get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// post performs a POST request with JSON body.
func (c *Client) post(path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// put performs a PUT request with JSON body.
func (c *Client) put(path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// delete performs a DELETE request.
func (c *Client) delete(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// Health checks if the server is reachable.
func (c *Client) Health() error {
	resp, err := c.get("/health")
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

// IsAuthError returns true if the error is an authentication error (401).
func IsAuthError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// parseError extracts an error message from an error response.
func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &errResp) == nil {
		msg := errResp.Error
		if msg == "" {
			msg = errResp.Message
		}
		if msg != "" {
			return &APIError{StatusCode: resp.StatusCode, Message: msg}
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("request failed with status %d", resp.StatusCode),
	}
}
