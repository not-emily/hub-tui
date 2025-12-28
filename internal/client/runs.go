package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Run represents a workflow run from hub-core.
type Run struct {
	ID        string    `json:"id"`
	Workflow  string    `json:"workflow_name"`
	Status    string    `json:"status"` // "running", "completed", "failed"
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"finished_at,omitempty"`
	Error     string    `json:"error,omitempty"`
	Result    *RunResult `json:"result,omitempty"`
}

// RunResult contains the workflow execution result.
type RunResult struct {
	WorkflowName string      `json:"workflow_name"`
	Success      bool        `json:"success"`
	Steps        []StepResult `json:"steps"`
	Error        string      `json:"error,omitempty"`
}

// StepResult contains the result of a single workflow step.
type StepResult struct {
	StepName string      `json:"step_name"`
	Success  bool        `json:"success"`
	Output   interface{} `json:"output,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// runsResponse is the API response wrapper.
type runsResponse struct {
	Active  []Run `json:"active"`
	History []Run `json:"history"`
}

// runResponse is the API response for a single run.
type runResponse struct {
	Run Run `json:"run"`
}

// runWorkflowResponse is the response from triggering a workflow.
type runWorkflowResponse struct {
	RunID string `json:"run_id"`
}

// RunWorkflow triggers a workflow and returns the run ID.
func (c *Client) RunWorkflow(name string) (string, error) {
	resp, err := c.post("/workflows/"+name+"/run", nil)
	if err != nil {
		return "", fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", parseError(resp)
	}

	var result runWorkflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("invalid response from server: %w", err)
	}

	return result.RunID, nil
}

// ListRuns fetches all runs from hub-core (active + history).
func (c *Client) ListRuns() ([]Run, error) {
	resp, err := c.get("/runs")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result runsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	// Combine active and history
	runs := append(result.Active, result.History...)
	return runs, nil
}

// GetRun fetches a specific run by ID.
func (c *Client) GetRun(id string) (*Run, error) {
	resp, err := c.get("/runs/" + id)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result runResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result.Run, nil
}

// CancelRun cancels a running workflow.
func (c *Client) CancelRun(id string) error {
	resp, err := c.post("/runs/"+id+"/cancel", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}

	return nil
}
