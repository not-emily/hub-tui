package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// TriggerConfig represents how a workflow is triggered.
type TriggerConfig struct {
	Type      string   `json:"type"`                // "schedule", "manual", "webhook", "condition"
	Cron      string   `json:"cron,omitempty"`      // cron expression for schedule type
	Frequency string   `json:"frequency,omitempty"` // "daily", "weekly", "monthly" (friendly)
	Time      string   `json:"time,omitempty"`      // "HH:MM" format
	Days      []string `json:"days,omitempty"`      // ["MON", "TUE", ...] for weekly
}

// WorkflowStep represents a single step in a workflow.
type WorkflowStep struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`              // "module", "integration", "utility", "primitive"
	Target  string                 `json:"target"`            // e.g., "notion.query_database"
	Profile string                 `json:"profile,omitempty"` // profile name for integrations
	Params  map[string]interface{} `json:"params,omitempty"`
	SaveAs  string                 `json:"save_as,omitempty"` // variable name to store output
}

// Workflow represents a workflow from hub-core.
type Workflow struct {
	Name                     string         `json:"name"`
	Description              string         `json:"description"`
	Trigger                  TriggerConfig  `json:"trigger"`
	Steps                    []WorkflowStep `json:"steps,omitempty"`
	Output                   string         `json:"output,omitempty"`                      // output variable reference
	NeedsAttentionOnComplete bool           `json:"needs_attention_on_complete,omitempty"` // if true, completed runs require attention
	Enabled                  bool           `json:"enabled"`
	NextRun                  *time.Time     `json:"next_run,omitempty"`                    // only for scheduled workflows
	Frequency                string         `json:"frequency,omitempty"`                   // human-readable schedule from API
}

// workflowRequest is the request body for create/update workflow API.
// This only includes fields the API accepts per the OpenAPI spec.
type workflowRequest struct {
	Name                     string            `json:"name"`
	Description              string            `json:"description,omitempty"`
	Trigger                  TriggerConfig     `json:"trigger"`
	Steps                    []workflowStepReq `json:"steps"`
	Output                   string            `json:"output,omitempty"`
	NeedsAttentionOnComplete bool              `json:"needs_attention_on_complete,omitempty"`
}

// workflowStepReq is a step in the API request format.
// Maps internal types to API-accepted types.
type workflowStepReq struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`              // "module", "utility", "llm"
	Target  string                 `json:"target,omitempty"`
	Profile string                 `json:"profile,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
	SaveAs  string                 `json:"save_as,omitempty"`
}

// toRequest converts a Workflow to the API request format.
func (wf *Workflow) toRequest() *workflowRequest {
	steps := make([]workflowStepReq, len(wf.Steps))
	for i, s := range wf.Steps {
		steps[i] = workflowStepReq{
			Name:    s.Name,
			Type:    mapStepType(s.Type),
			Target:  s.Target,
			Profile: s.Profile,
			Params:  s.Params,
			SaveAs:  s.SaveAs,
		}
	}
	return &workflowRequest{
		Name:                     wf.Name,
		Description:              wf.Description,
		Trigger:                  wf.Trigger,
		Steps:                    steps,
		Output:                   wf.Output,
		NeedsAttentionOnComplete: wf.NeedsAttentionOnComplete,
	}
}

// mapStepType converts internal step types to API-accepted types.
func mapStepType(t string) string {
	switch t {
	case "integration":
		// Integrations are treated as modules by the API
		return "module"
	case "primitive":
		// Primitives are treated as utilities by the API
		return "utility"
	case "module", "utility", "llm":
		return t
	default:
		return "module"
	}
}

// workflowsResponse is the API response wrapper for listing workflows.
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

// GetWorkflow fetches a single workflow by name.
func (c *Client) GetWorkflow(name string) (*Workflow, error) {
	resp, err := c.get("/workflows/" + name)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var wf Workflow
	if err := json.NewDecoder(resp.Body).Decode(&wf); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &wf, nil
}

// CreateWorkflow creates a new workflow.
func (c *Client) CreateWorkflow(wf *Workflow) error {
	req := wf.toRequest()
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode workflow: %w", err)
	}

	resp, err := c.post("/workflows", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return parseError(resp)
	}

	return nil
}

// UpdateWorkflow updates an existing workflow.
func (c *Client) UpdateWorkflow(name string, wf *Workflow) error {
	req := wf.toRequest()
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode workflow: %w", err)
	}

	resp, err := c.put("/workflows/"+name, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}

	return nil
}

// DeleteWorkflow deletes a workflow.
func (c *Client) DeleteWorkflow(name string) error {
	resp, err := c.delete("/workflows/" + name)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}

	return nil
}

// --- Workflow Builder API ---

// Tool represents an available tool from the builder API.
type Tool struct {
	Target            string      `json:"target"`
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	Params            []ToolParam `json:"params"`
	OutputDescription string      `json:"output_description,omitempty"`
	RequiresProfile   bool        `json:"requires_profile,omitempty"`
}

// ToolParam represents a parameter for a tool.
type ToolParam struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // "string", "number", "boolean", "array", "object"
	Required    bool        `json:"required"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Example     interface{} `json:"example,omitempty"`    // Sample value showing expected format
	Properties  []ToolParam `json:"properties,omitempty"` // Nested params for object types
	Items       *ToolParam  `json:"items,omitempty"`      // Element schema for array types
}

// ToolsResponse is the response from GET /workflows/builder/tools.
type ToolsResponse struct {
	Tools ToolCategories `json:"tools"`
}

// ToolCategories groups tools by their type.
type ToolCategories struct {
	Modules      map[string][]Tool `json:"modules"`
	Integrations map[string][]Tool `json:"integrations"`
	Utilities    map[string][]Tool `json:"utilities"`
	Primitives   map[string][]Tool `json:"primitives"`
}

// GetBuilderTools fetches all available tools for the workflow builder.
func (c *Client) GetBuilderTools() (*ToolsResponse, error) {
	resp, err := c.get("/workflows/builder/tools")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result ToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}

// StepTestRequest is the request body for testing a step.
type StepTestRequest struct {
	Step      WorkflowStep           `json:"step"`
	Variables map[string]interface{} `json:"variables"`
}

// StepTestResult is the response from testing a step.
type StepTestResult struct {
	Success bool        `json:"success"`
	Output  interface{} `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// TestStep tests a single workflow step.
func (c *Client) TestStep(req *StepTestRequest) (*StepTestResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.post("/workflows/builder/steps/test", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result StepTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}

// ScheduleRequest is the request body for previewing a schedule.
type ScheduleRequest struct {
	Frequency string   `json:"frequency"` // "daily", "weekly", "monthly"
	Time      string   `json:"time"`      // "HH:MM"
	Days      []string `json:"days,omitempty"`
}

// SchedulePreview is the response from previewing a schedule.
type SchedulePreview struct {
	Cron        string   `json:"cron"`
	Description string   `json:"description"`
	NextRuns    []string `json:"next_runs"`
}

// PreviewSchedule converts a friendly schedule to cron and shows next runs.
func (c *Client) PreviewSchedule(req *ScheduleRequest) (*SchedulePreview, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.post("/workflows/builder/schedule/preview", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result SchedulePreview
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}

// TransformRequest is the request body for previewing a transform.
type TransformRequest struct {
	Operation string                 `json:"operation"` // "filter", "pick", "sort", "first", "last", "count"
	Params    map[string]interface{} `json:"params"`
}

// TransformPreview is the response from previewing a transform.
type TransformPreview struct {
	Step WorkflowStep `json:"step"`
}

// PreviewTransform generates a workflow step from a transform preset.
func (c *Client) PreviewTransform(req *TransformRequest) (*TransformPreview, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.post("/workflows/builder/transform/preview", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result TransformPreview
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}

// ValidationError represents a single validation error.
type ValidationError struct {
	Step    *int   `json:"step,omitempty"` // nil for workflow-level errors
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult is the response from validating a workflow.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidateWorkflow validates a workflow definition.
func (c *Client) ValidateWorkflow(wf *Workflow) (*ValidationResult, error) {
	body, err := json.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("failed to encode workflow: %w", err)
	}

	resp, err := c.post("/workflows/builder/validate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result ValidationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &result, nil
}
