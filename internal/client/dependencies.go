package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Dependency represents a CLI dependency and its status.
type Dependency struct {
	Name             string `json:"name"`
	Required         bool   `json:"required"`
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installed_version"`
	RequiredVersion  string `json:"required_version"`
	NeedsUpdate      bool   `json:"needs_update"`
	Error            string `json:"error,omitempty"`
}

// DependencyUpdate represents an available dependency update.
type DependencyUpdate struct {
	Integration    string `json:"integration"`
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
}

// DependencyInstallRequest is the request body for installing a dependency.
type DependencyInstallRequest struct {
	Version string `json:"version"`
}

// DependencyInstallResponse is the response from installing a dependency.
type DependencyInstallResponse struct {
	Success bool       `json:"success"`
	Status  Dependency `json:"status"`
}

// DependencyUpdatesResponse is the response from checking for dependency updates.
type DependencyUpdatesResponse struct {
	UpdatesAvailable bool               `json:"updates_available"`
	Updates          []DependencyUpdate `json:"updates"`
}

// HubUpdateInfo contains information about available hub-core updates.
type HubUpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
	DownloadURL     string `json:"download_url"`
}

// HubUpdateResponse is the response from applying a hub-core update.
type HubUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetDependencies fetches CLI dependencies and their status.
// If integration is non-empty, filters to only that integration's dependencies.
// Requires admin permissions.
func (c *Client) GetDependencies(integration string) ([]Dependency, error) {
	path := "/admin/dependencies"
	if integration != "" {
		path += "?integration=" + integration
	}

	resp, err := c.get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result struct {
		Dependencies []Dependency `json:"dependencies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return result.Dependencies, nil
}

// InstallDependency installs or updates a CLI dependency.
// Requires admin permissions.
func (c *Client) InstallDependency(name, version string) (*DependencyInstallResponse, error) {
	reqBody, err := json.Marshal(DependencyInstallRequest{Version: version})
	if err != nil {
		return nil, err
	}

	resp, err := c.post(fmt.Sprintf("/admin/dependencies/%s/install", name), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result DependencyInstallResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &result, nil
}

// CheckDependencyUpdates checks for available updates to all dependencies.
// Requires admin permissions.
func (c *Client) CheckDependencyUpdates() (*DependencyUpdatesResponse, error) {
	resp, err := c.post("/admin/dependencies/check", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result DependencyUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &result, nil
}

// GetHubUpdates checks for available hub-core updates.
// Returns 404 error if repository is private.
// Requires admin permissions.
func (c *Client) GetHubUpdates() (*HubUpdateInfo, error) {
	resp, err := c.get("/admin/hub/updates")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 404 means repo is private (updates not available)
	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{
			StatusCode: 404,
			Message:    "hub updates not available (repository is private)",
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result HubUpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &result, nil
}

// ApplyHubUpdate applies a hub-core update and restarts the service.
// The server will restart after this call, causing a temporary disconnection.
// Requires admin permissions.
func (c *Client) ApplyHubUpdate() (*HubUpdateResponse, error) {
	resp, err := c.post("/admin/hub/updates/apply", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result HubUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &result, nil
}
