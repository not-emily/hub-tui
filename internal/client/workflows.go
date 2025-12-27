package client

import (
	"encoding/json"
	"fmt"
)

// Workflow represents a workflow from hub-core.
type Workflow struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// workflowsResponse is the API response wrapper.
type workflowsResponse struct {
	Workflows []Workflow `json:"workflows"`
}

// ListWorkflows fetches all workflows from hub-core.
func (c *Client) ListWorkflows() ([]Workflow, error) {
	resp, err := c.get("/workflows")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result workflowsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Workflows, nil
}
