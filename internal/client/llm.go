package client

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// LLMProfile represents an LLM profile configuration.
type LLMProfile struct {
	Integration string `json:"integration"`
	Profile     string `json:"profile,omitempty"`
	Model       string `json:"model"`
}

// LLMProfileList is the response from GET /llm/profiles.
type LLMProfileList struct {
	Profiles       map[string]LLMProfile `json:"profiles"`
	DefaultProfile string                `json:"default_profile"`
}

// LLMTestResult is the response from POST /llm/profiles/{name}/test.
type LLMTestResult struct {
	Success   bool   `json:"success"`
	Model     string `json:"model"`
	LatencyMs int    `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// LLMProfileConfig is the request body for PUT /llm/profiles/{name}.
type LLMProfileConfig struct {
	Name        string `json:"name,omitempty"`
	Integration string `json:"integration"`
	Profile     string `json:"profile,omitempty"`
	Model       string `json:"model"`
}

// ListLLMProfiles fetches all LLM profiles from hub-core.
func (c *Client) ListLLMProfiles() (*LLMProfileList, error) {
	resp, err := c.get("/llm/profiles")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result LLMProfileList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}

// CreateLLMProfile creates a new LLM profile.
func (c *Client) CreateLLMProfile(name string, config LLMProfileConfig) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	resp, err := c.put("/llm/profiles/"+name, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// UpdateLLMProfile updates an existing LLM profile.
// If config.Name is set and different from name, the profile will be renamed.
func (c *Client) UpdateLLMProfile(name string, config LLMProfileConfig) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	resp, err := c.put("/llm/profiles/"+name, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// DeleteLLMProfile deletes an LLM profile.
func (c *Client) DeleteLLMProfile(name string) error {
	resp, err := c.delete("/llm/profiles/" + name)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// TestLLMProfile tests an LLM profile's connectivity.
func (c *Client) TestLLMProfile(name string) (*LLMTestResult, error) {
	resp, err := c.post("/llm/profiles/"+name+"/test", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result LLMTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}

// SetDefaultLLMProfile sets the default LLM profile.
func (c *Client) SetDefaultLLMProfile(name string) error {
	req := struct {
		Profile string `json:"profile"`
	}{
		Profile: name,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.put("/llm/default", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}
