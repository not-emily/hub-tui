package client

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Module represents a module from hub-core.
type Module struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Version     string   `json:"version"`
	Tools       []string `json:"tools"`
}

// AvailableModule represents a module from the registry.
type AvailableModule struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	ReleaseTag       string   `json:"release_tag"`
	Description      string   `json:"description"`
	Keywords         []string `json:"keywords"`
	Installed        bool     `json:"installed"`
	InstalledVersion string   `json:"installed_version"`
	UpdateAvailable  bool     `json:"update_available"`
}

// UninstallResult is the response from attempting to uninstall a module.
type UninstallResult struct {
	Success         bool     `json:"success"`
	Module          string   `json:"module"`
	Error           string   `json:"error"`
	Message         string   `json:"message"`
	AffectedUsers   []string `json:"affected_users"`
	ConfirmRequired bool     `json:"confirm_required"`
}

// UpdateResult is the response from updating a module.
type UpdateResult struct {
	Success         bool   `json:"success"`
	Module          string `json:"module"`
	PreviousVersion string `json:"previous_version"`
	NewVersion      string `json:"new_version"`
	AlreadyLatest   bool   `json:"already_latest"`
	CurrentVersion  string `json:"current_version"`
}

// modulesResponse is the API response wrapper.
type modulesResponse struct {
	Modules []Module `json:"modules"`
}

// availableModulesResponse is the API response wrapper for available modules.
type availableModulesResponse struct {
	Modules []AvailableModule `json:"modules"`
}

// installModuleRequest is the request body for installing a module.
type installModuleRequest struct {
	Name string `json:"name"`
}

// ListModules fetches all modules from hub-core.
func (c *Client) ListModules() ([]Module, error) {
	resp, err := c.get("/modules")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result modulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Modules, nil
}

// EnableModule enables a module.
func (c *Client) EnableModule(name string) error {
	resp, err := c.post("/modules/"+name+"/enable", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// DisableModule disables a module.
func (c *Client) DisableModule(name string) error {
	resp, err := c.post("/modules/"+name+"/disable", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

// ListAvailableModules fetches modules from the registry (admin only).
func (c *Client) ListAvailableModules() ([]AvailableModule, error) {
	resp, err := c.get("/modules/available")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result availableModulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Modules, nil
}

// InstallModule installs a module from the registry (admin only).
func (c *Client) InstallModule(name string) error {
	reqBody, err := json.Marshal(installModuleRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.post("/modules/install", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return parseError(resp)
	}

	return nil
}

// UninstallModule attempts to uninstall a module (admin only).
// Returns UninstallResult which may indicate users are affected and confirmation is required.
func (c *Client) UninstallModule(name string) (*UninstallResult, error) {
	resp, err := c.delete("/modules/" + name)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response body regardless of status code
	// The API returns a result even when confirmation is required
	var result UninstallResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != 200 {
			return nil, parseError(resp)
		}
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}

// UninstallModuleForce uninstalls a module even if users have it enabled (admin only).
func (c *Client) UninstallModuleForce(name string) error {
	resp, err := c.delete("/modules/" + name + "?force=true")
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return parseError(resp)
	}

	return nil
}

// UpdateModule updates a module to the latest version (admin only).
func (c *Client) UpdateModule(name string) (*UpdateResult, error) {
	resp, err := c.post("/modules/"+name+"/update", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result UpdateResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}
