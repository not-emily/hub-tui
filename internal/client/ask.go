package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Response status constants.
const (
	StatusNeedsInput = "needs_input"
	StatusExecuted   = "executed"
	StatusError      = "error"
)

// AskRequest supports both natural language input and structured params.
type AskRequest struct {
	Input  string                 `json:"input,omitempty"`
	Target string                 `json:"target,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// AskResponse is the status-based response from /ask endpoints.
type AskResponse struct {
	// New status-based fields
	Status string         `json:"status,omitempty"` // needs_input, executed, error
	Target string         `json:"target,omitempty"`
	Schema *ParamSchema   `json:"schema,omitempty"`
	Result *ExecuteResult `json:"result,omitempty"`
	Error  *AskError      `json:"error,omitempty"`

	// Legacy fields for backward compatibility with streaming responses
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ParamSchema describes the form schema for parameter collection.
type ParamSchema struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Params      []ParamField `json:"params"`
}

// ParamField describes a single parameter in the schema.
type ParamField struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, array, object
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// ExecuteResult contains the result of a successful execution.
type ExecuteResult struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// AskError contains error details for failed requests.
type AskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RouteInfo contains routing information from the route event.
type RouteInfo struct {
	Type   string `json:"type"`   // "assistant", "workflow", "module", etc.
	Target string `json:"target"` // Name of the target (e.g., "fitness_trainer")
}

// AskCallbacks contains callbacks for SSE events.
type AskCallbacks struct {
	OnRoute func(RouteInfo) // Called when route event received
	OnChunk func(string)    // Called for each content chunk
}

// Ask sends a message to the /ask endpoint and streams the response.
func (c *Client) Ask(ctx context.Context, message string, callbacks AskCallbacks) (*AskResponse, error) {
	reqBody, err := json.Marshal(AskRequest{Input: message})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ask", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	// Check if streaming response (SSE)
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return c.readSSEStream(ctx, resp, callbacks)
	}

	// Non-streaming response - read entire body
	var apiResp AskResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	if callbacks.OnChunk != nil {
		callbacks.OnChunk(apiResp.Message)
	}

	return &apiResp, nil
}

// readSSEStream reads a Server-Sent Events stream with typed events.
func (c *Client) readSSEStream(ctx context.Context, resp *http.Response, callbacks AskCallbacks) (*AskResponse, error) {
	var fullContent strings.Builder
	var currentEvent string
	var result AskResponse

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return &AskResponse{Message: fullContent.String()}, ctx.Err()
		default:
		}

		line := scanner.Text()

		// Parse event type
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		// Parse data
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "route":
				var route RouteInfo
				if err := json.Unmarshal([]byte(data), &route); err == nil {
					if callbacks.OnRoute != nil {
						callbacks.OnRoute(route)
					}
				}

			case "chunk":
				var chunk struct {
					Content string `json:"content"`
				}
				if err := json.Unmarshal([]byte(data), &chunk); err == nil {
					if chunk.Content != "" {
						if callbacks.OnChunk != nil {
							callbacks.OnChunk(chunk.Content)
						}
						fullContent.WriteString(chunk.Content)
					}
				}

			case "needs_input":
				// Parameter collection required
				var resp AskResponse
				if err := json.Unmarshal([]byte(data), &resp); err == nil {
					result.Status = resp.Status
					result.Target = resp.Target
					result.Schema = resp.Schema
				}

			case "executed":
				// Operation completed successfully
				var resp AskResponse
				if err := json.Unmarshal([]byte(data), &resp); err == nil {
					result.Status = resp.Status
					result.Target = resp.Target
					result.Result = resp.Result
					if resp.Result != nil {
						result.Message = resp.Result.Message
						result.Success = resp.Result.Success
					}
				}

			case "error":
				// Operation failed
				var resp AskResponse
				if err := json.Unmarshal([]byte(data), &resp); err == nil {
					result.Status = resp.Status
					result.Target = resp.Target
					result.Error = resp.Error
				}

			case "done":
				// Parse the full response structure (supports both old and new formats)
				var done AskResponse
				if err := json.Unmarshal([]byte(data), &done); err == nil {
					// Copy all fields to result
					result.Status = done.Status
					result.Target = done.Target
					result.Schema = done.Schema
					result.Result = done.Result
					result.Error = done.Error
					result.Success = done.Success
					result.Message = done.Message

					// For status-based responses, populate legacy fields
					if done.Status == StatusExecuted && done.Result != nil {
						if result.Message == "" {
							result.Message = done.Result.Message
						}
						result.Success = done.Result.Success
					}

					// For non-streaming responses (utility, module, workflow, unknown),
					// the message comes in done event, not chunks
					if result.Message != "" && fullContent.Len() == 0 {
						if callbacks.OnChunk != nil {
							callbacks.OnChunk(result.Message)
						}
						fullContent.WriteString(result.Message)
					}
				}
			}

			currentEvent = "" // Reset for next event
		}
	}

	if err := scanner.Err(); err != nil {
		return &AskResponse{Message: fullContent.String()}, err
	}

	// Use accumulated content if message not set
	if result.Message == "" {
		result.Message = fullContent.String()
	}

	return &result, nil
}

// AskDirect sends a blocking request to /ask/direct.
// Used for both natural language queries and structured param submission.
func (c *Client) AskDirect(req AskRequest) (*AskResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/ask/direct", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result AskResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	// For backward compatibility, populate Message from Result if status is executed
	if result.Status == StatusExecuted && result.Result != nil && result.Message == "" {
		result.Message = result.Result.Message
		result.Success = result.Result.Success
	}

	return &result, nil
}
