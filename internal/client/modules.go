package client

import (
	"encoding/json"
	"fmt"
)

// Module represents a module from hub-core.
type Module struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Version     string `json:"version"`
}

// modulesResponse is the API response wrapper.
type modulesResponse struct {
	Modules []Module `json:"modules"`
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
