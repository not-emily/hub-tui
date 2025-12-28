package client

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Integration represents an integration from hub-core.
type Integration struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Configured     bool     `json:"configured"`
	Profiles       []string `json:"profiles"`        // Configured profile names
	DefaultProfile string   `json:"default_profile"` // Default profile to use
	Fields         []string `json:"fields"`          // Required config fields
}

// integrationsResponse is the API response wrapper.
type integrationsResponse struct {
	Integrations []Integration `json:"integrations"`
}

// ListIntegrations fetches all integrations from hub-core.
func (c *Client) ListIntegrations() ([]Integration, error) {
	resp, err := c.get("/integrations")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result integrationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Integrations, nil
}

// configureRequest is the request body for configuring an integration.
type configureRequest struct {
	Profile string            `json:"profile"`
	Config  map[string]string `json:"config"`
}

// ConfigureIntegration configures an integration profile.
func (c *Client) ConfigureIntegration(name, profile string, config map[string]string) error {
	req := configureRequest{
		Profile: profile,
		Config:  config,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	resp, err := c.post("/integrations/"+name+"/configure", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// TestIntegration tests an integration.
func (c *Client) TestIntegration(name string) error {
	resp, err := c.post("/integrations/"+name+"/test", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}
