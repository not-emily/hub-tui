package client

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// LLM config type - Provider/Account/Profile model
//
// This file contains client methods for integrations with config_type: "llm".
// All methods take an integration name parameter to support multiple LLM integrations.

// ProviderAccount represents a configured provider with its accounts.
type ProviderAccount struct {
	Provider    string   `json:"provider"`     // e.g., "openai", "anthropic"
	DisplayName string   `json:"display_name"` // e.g., "OpenAI"
	Accounts    []string `json:"accounts"`     // e.g., ["default", "work"]
}

// AvailableProvider represents a provider that the integration supports.
type AvailableProvider struct {
	Name        string `json:"name"`         // e.g., "openai"
	DisplayName string `json:"display_name"` // e.g., "OpenAI"
}

// ProviderFieldInfo describes a configuration field required by a provider.
type ProviderFieldInfo struct {
	Key      string `json:"key"`      // Field identifier (e.g., "api_key", "base_url")
	Label    string `json:"label"`    // Human-readable label for UI
	Required bool   `json:"required"` // Whether field must be provided
	Secret   bool   `json:"secret"`   // Whether to mask input (passwords, API keys)
	Default  string `json:"default"`  // Default value if not provided
}

// LLMProfile represents an LLM profile configuration.
type LLMProfile struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Account   string `json:"account"`
	Model     string `json:"model"`
	IsDefault bool   `json:"is_default"`
}

// LLMProfileList is the response from listing LLM profiles.
type LLMProfileList struct {
	Profiles []LLMProfile `json:"profiles"`
}

// AddProviderRequest is the request body for adding a provider account.
type AddProviderRequest struct {
	Provider string            `json:"provider"`
	Account  string            `json:"account"`
	Fields   map[string]string `json:"fields"`
}

// CreateProfileRequest is the request body for creating an LLM profile.
type CreateProfileRequest struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Account  string `json:"account"`
	Model    string `json:"model"`
}

// LLMTestResult is the response from testing an LLM profile.
type LLMTestResult struct {
	Success   bool   `json:"success"`
	Model     string `json:"model"`
	LatencyMs int    `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// --- Provider Methods ---

// providersResponse is the API response for listing providers.
type providersResponse struct {
	Providers []ProviderAccount `json:"providers"`
}

// ListLLMProviders fetches configured providers for an LLM integration.
func (c *Client) ListLLMProviders(integration string) ([]ProviderAccount, error) {
	resp, err := c.get("/integrations/" + integration + "/providers")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result providersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Providers, nil
}

// availableProvidersResponse is the API response for listing available providers.
type availableProvidersResponse struct {
	Providers []AvailableProvider `json:"providers"`
}

// ListAvailableLLMProviders fetches all providers that an integration supports.
func (c *Client) ListAvailableLLMProviders(integration string) ([]AvailableProvider, error) {
	resp, err := c.get("/integrations/" + integration + "/providers/available")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result availableProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Providers, nil
}

// providerFieldsResponse is the API response for provider fields.
type providerFieldsResponse struct {
	Fields []ProviderFieldInfo `json:"fields"`
}

// GetLLMProviderFields fetches field requirements for a provider.
func (c *Client) GetLLMProviderFields(integration, provider string) ([]ProviderFieldInfo, error) {
	resp, err := c.get("/integrations/" + integration + "/providers/" + provider + "/fields")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result providerFieldsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Fields, nil
}

// AddLLMProvider adds a new provider account to an LLM integration.
func (c *Client) AddLLMProvider(integration string, req AddProviderRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.post("/integrations/"+integration+"/providers", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// DeleteLLMProvider removes a provider account from an LLM integration.
func (c *Client) DeleteLLMProvider(integration, provider, account string) error {
	resp, err := c.delete("/integrations/" + integration + "/providers/" + provider + "/" + account)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// --- Profile Methods ---

// profilesResponse is the API response for listing profiles.
type profilesResponse struct {
	Profiles []LLMProfile `json:"profiles"`
}

// ListLLMProfiles fetches all LLM profiles for an integration.
func (c *Client) ListLLMProfiles(integration string) (*LLMProfileList, error) {
	resp, err := c.get("/integrations/" + integration + "/profiles")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result profilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &LLMProfileList{Profiles: result.Profiles}, nil
}

// CreateLLMProfile creates a new LLM profile.
func (c *Client) CreateLLMProfile(integration string, req CreateProfileRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.post("/integrations/"+integration+"/profiles", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return parseError(resp)
	}
	return nil
}

// DeleteLLMProfile deletes an LLM profile.
func (c *Client) DeleteLLMProfile(integration, profile string) error {
	resp, err := c.delete("/integrations/" + integration + "/profiles/" + profile)
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
func (c *Client) TestLLMProfile(integration, profile string) (*LLMTestResult, error) {
	resp, err := c.post("/integrations/"+integration+"/profiles/"+profile+"/test", nil)
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

// LLMModelsResult contains the paginated models response.
type LLMModelsResult struct {
	Models     []ModelInfo
	Pagination ModelsPagination
}

// llmModelsResponse is the API response for listing LLM models.
type llmModelsResponse struct {
	Models     []ModelInfo      `json:"models"`
	Pagination ModelsPagination `json:"pagination"`
}

// ListLLMModels fetches available models for an LLM provider with pagination.
func (c *Client) ListLLMModels(integration, provider string, limit int, cursor string) (*LLMModelsResult, error) {
	path := fmt.Sprintf("/integrations/%s/models?provider=%s&limit=%d", integration, provider, limit)
	if cursor != "" {
		path += "&cursor=" + cursor
	}

	resp, err := c.get(path)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result llmModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &LLMModelsResult{
		Models:     result.Models,
		Pagination: result.Pagination,
	}, nil
}

// SetDefaultLLMProfile sets the default LLM profile for an integration.
func (c *Client) SetDefaultLLMProfile(integration, profile string) error {
	req := struct {
		Profile string `json:"profile"`
	}{
		Profile: profile,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.put("/integrations/"+integration+"/profiles/set-default", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}
