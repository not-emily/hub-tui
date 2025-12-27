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

// AskRequest is the request body for /ask.
type AskRequest struct {
	Input string `json:"input"`
}

// AskResponse is the final response from /ask.
type AskResponse struct {
	Success bool   `json:"success"`
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

			case "done":
				var done struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal([]byte(data), &done); err == nil {
					result.Success = done.Success
					result.Message = done.Message
					// For non-streaming responses (utility, module, workflow, unknown),
					// the message comes in done event, not chunks
					if done.Message != "" && fullContent.Len() == 0 {
						if callbacks.OnChunk != nil {
							callbacks.OnChunk(done.Message)
						}
						fullContent.WriteString(done.Message)
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
