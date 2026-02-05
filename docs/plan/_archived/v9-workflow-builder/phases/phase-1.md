# Phase 1: Client Layer

> **Depends on:** None
> **Enables:** Phase 2 (List View Enhancement)
>
> See: [Full Plan](../plan.md)

## Goal

Extend the client layer with full workflow CRUD operations and all workflow builder API methods.

## Key Deliverables

- Extended `Workflow` struct with `Steps` and `Output` fields
- `WorkflowStep` struct matching API schema
- CRUD methods: `GetWorkflow`, `CreateWorkflow`, `UpdateWorkflow`, `DeleteWorkflow`
- Builder API methods: `GetBuilderTools`, `TestStep`, `PreviewSchedule`, `PreviewTransform`, `ValidateWorkflow`
- Response types for all builder endpoints

## Files to Modify/Create

- `internal/client/workflows.go` — Extend existing with full structs and CRUD methods

## Implementation Notes

### Extend Workflow Struct

The existing `Workflow` struct has basic fields. Extend it:

```go
type WorkflowStep struct {
    Name    string                 `json:"name"`
    Type    string                 `json:"type"`    // "module", "integration", "utility", "primitive"
    Target  string                 `json:"target"`  // e.g., "notion.query_database"
    Profile string                 `json:"profile,omitempty"`
    Params  map[string]interface{} `json:"params,omitempty"`
    SaveAs  string                 `json:"save_as,omitempty"`
}

type TriggerConfig struct {
    Type      string   `json:"type"`                // "schedule", "manual"
    Cron      string   `json:"cron,omitempty"`
    Frequency string   `json:"frequency,omitempty"` // "daily", "weekly", "monthly"
    Time      string   `json:"time,omitempty"`      // "08:30"
    Days      []string `json:"days,omitempty"`      // ["MON", "FRI"]
}

// Extend existing Workflow struct
type Workflow struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Trigger     TriggerConfig  `json:"trigger"`
    Steps       []WorkflowStep `json:"steps"`
    Output      string         `json:"output,omitempty"`
    Enabled     bool           `json:"enabled"`
    NextRun     *time.Time     `json:"next_run,omitempty"`
    Frequency   string         `json:"frequency,omitempty"`
}
```

### CRUD Methods

```go
// GetWorkflow fetches a single workflow by name
func (c *Client) GetWorkflow(name string) (*Workflow, error)
// Endpoint: GET /workflows/{name}

// CreateWorkflow creates a new workflow
func (c *Client) CreateWorkflow(wf *Workflow) error
// Endpoint: POST /workflows

// UpdateWorkflow updates an existing workflow
func (c *Client) UpdateWorkflow(name string, wf *Workflow) error
// Endpoint: PUT /workflows/{name}

// DeleteWorkflow deletes a workflow
func (c *Client) DeleteWorkflow(name string) error
// Endpoint: DELETE /workflows/{name}
```

### Builder API Methods

```go
// ToolsResponse from GET /workflows/builder/tools
type ToolsResponse struct {
    Tools struct {
        Modules      map[string][]Tool `json:"modules"`
        Integrations map[string][]Tool `json:"integrations"`
        Utilities    map[string][]Tool `json:"utilities"`
        Primitives   map[string][]Tool `json:"primitives"`
    } `json:"tools"`
}

type Tool struct {
    Target            string      `json:"target"`
    Name              string      `json:"name"`
    Description       string      `json:"description"`
    Params            []ToolParam `json:"params"`
    OutputDescription string      `json:"output_description,omitempty"`
    RequiresProfile   bool        `json:"requires_profile,omitempty"`
}

type ToolParam struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"`  // "string", "number", "boolean", "array", "object"
    Required    bool        `json:"required"`
    Description string      `json:"description,omitempty"`
    Default     interface{} `json:"default,omitempty"`
}

// GetBuilderTools fetches all available tools
func (c *Client) GetBuilderTools() (*ToolsResponse, error)
// Endpoint: GET /workflows/builder/tools

// StepTestRequest for testing a step
type StepTestRequest struct {
    Step      WorkflowStep           `json:"step"`
    Variables map[string]interface{} `json:"variables"`
}

type StepTestResult struct {
    Success bool        `json:"success"`
    Output  interface{} `json:"output,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func (c *Client) TestStep(req *StepTestRequest) (*StepTestResult, error)
// Endpoint: POST /workflows/builder/steps/test

// ScheduleRequest for previewing a schedule
type ScheduleRequest struct {
    Frequency string   `json:"frequency"` // "daily", "weekly", "monthly"
    Time      string   `json:"time"`      // "08:30"
    Days      []string `json:"days,omitempty"`
}

type SchedulePreview struct {
    Cron        string   `json:"cron"`
    Description string   `json:"description"`
    NextRuns    []string `json:"next_runs"`
}

func (c *Client) PreviewSchedule(req *ScheduleRequest) (*SchedulePreview, error)
// Endpoint: POST /workflows/builder/schedule/preview

// TransformRequest for previewing a transform
type TransformRequest struct {
    Operation string                 `json:"operation"` // "filter", "pick", "sort", etc.
    Params    map[string]interface{} `json:"params"`
}

type TransformPreview struct {
    Step WorkflowStep `json:"step"`
}

func (c *Client) PreviewTransform(req *TransformRequest) (*TransformPreview, error)
// Endpoint: POST /workflows/builder/transform/preview

// ValidationResult from validating a workflow
type ValidationError struct {
    Step    *int   `json:"step,omitempty"` // nil for workflow-level errors
    Field   string `json:"field"`
    Message string `json:"message"`
}

type ValidationResult struct {
    Valid  bool              `json:"valid"`
    Errors []ValidationError `json:"errors,omitempty"`
}

func (c *Client) ValidateWorkflow(wf *Workflow) (*ValidationResult, error)
// Endpoint: POST /workflows/builder/validate
```

### Integration Profiles

Check if `GetIntegrationProfiles` already exists. If not, add:

```go
func (c *Client) GetIntegrationProfiles(name string) ([]string, error)
// Endpoint: GET /integrations/{name}/profiles
// Returns profile names as string slice
```

## Validation

- [ ] All new types compile without errors
- [ ] `GetWorkflow("existing-workflow")` returns full workflow with steps
- [ ] `CreateWorkflow` successfully creates a workflow (verify via list)
- [ ] `UpdateWorkflow` successfully updates (verify changes persist)
- [ ] `DeleteWorkflow` removes workflow from list
- [ ] `GetBuilderTools` returns categorized tools with schemas
- [ ] `TestStep` with a simple primitive (e.g., `time.now`) returns output
- [ ] `PreviewSchedule` returns valid cron and next runs
- [ ] `PreviewTransform` with filter operation returns generated step
- [ ] `ValidateWorkflow` with invalid workflow returns errors
