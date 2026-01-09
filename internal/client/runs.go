package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Run represents a workflow run from hub-core.
type Run struct {
	ID             string     `json:"id"`
	Workflow       string     `json:"workflow_name"`
	Status         string     `json:"status"` // "running", "completed", "failed"
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        time.Time  `json:"finished_at,omitempty"`
	Error          string     `json:"error,omitempty"`
	Result         *RunResult `json:"result,omitempty"`
	NeedsAttention bool       `json:"needs_attention"`
}

// RunResult contains the workflow execution result.
type RunResult struct {
	WorkflowName string       `json:"workflow_name"`
	Success      bool         `json:"success"`
	Output       string       `json:"output,omitempty"`
	Steps        []StepResult `json:"steps"`
	Error        string       `json:"error,omitempty"`
}

// StepResult contains the result of a single workflow step.
type StepResult struct {
	StepName string      `json:"step_name"`
	Success  bool        `json:"success"`
	Output   interface{} `json:"output,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// Pagination contains pagination info from the API.
type Pagination struct {
	Total      int    `json:"total"`
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// RunsResponse contains runs and pagination info.
type RunsResponse struct {
	Runs       []Run      `json:"runs"`
	Pagination Pagination `json:"pagination"`
}

// RunsFilter contains optional query parameters for listing runs.
type RunsFilter struct {
	Limit          int    // Max results (1-100, default 50)
	Cursor         string // Run ID to start after (for pagination)
	Status         string // Filter: pending, running, completed, failed, cancelled
	Since          string // Filter: runs started on/after date (YYYY-MM-DD)
	Until          string // Filter: runs started before date (YYYY-MM-DD)
	NeedsAttention *bool  // Filter: true or false (nil = no filter)
}

// runsResponse is the API response wrapper.
type runsResponse struct {
	Runs       []Run      `json:"runs"`
	Pagination Pagination `json:"pagination"`
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

// ListRuns fetches runs from hub-core with optional filtering.
func (c *Client) ListRuns(filter *RunsFilter) (*RunsResponse, error) {
	path := "/runs"
	if filter != nil {
		path += buildRunsQuery(filter)
	}

	resp, err := c.get(path)
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

	return &RunsResponse{
		Runs:       result.Runs,
		Pagination: result.Pagination,
	}, nil
}

// buildRunsQuery builds query string from filter options.
func buildRunsQuery(f *RunsFilter) string {
	params := url.Values{}
	if f.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", f.Limit))
	}
	if f.Cursor != "" {
		params.Set("cursor", f.Cursor)
	}
	if f.Status != "" {
		params.Set("status", f.Status)
	}
	if f.Since != "" {
		params.Set("since", f.Since)
	}
	if f.Until != "" {
		params.Set("until", f.Until)
	}
	if f.NeedsAttention != nil {
		params.Set("needs_attention", fmt.Sprintf("%t", *f.NeedsAttention))
	}
	if len(params) == 0 {
		return ""
	}
	return "?" + params.Encode()
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

	// API returns the run directly, not wrapped
	var run Run
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &run, nil
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

// DismissRun dismisses a run that needs attention.
func (c *Client) DismissRun(id string) error {
	resp, err := c.post("/runs/"+id+"/dismiss", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}

	return nil
}
