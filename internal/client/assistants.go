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

// Assistant represents an assistant from hub-core.
type Assistant struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// assistantsResponse is the API response wrapper.
type assistantsResponse struct {
	Assistants []Assistant `json:"assistants"`
}

// ListAssistants fetches all assistants from hub-core.
func (c *Client) ListAssistants() ([]Assistant, error) {
	resp, err := c.get("/assistants")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result assistantsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Assistants, nil
}

// AssistantChatRequest is the request body for /assistants/{name}/chat.
type AssistantChatRequest struct {
	Message string `json:"message"`
}

// AssistantInfo contains info from the assistant event.
type AssistantInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// AssistantChatCallbacks contains callbacks for assistant chat SSE events.
type AssistantChatCallbacks struct {
	OnAssistant func(AssistantInfo) // Called when assistant event received
	OnChunk     func(string)        // Called for each content chunk
}

// AssistantChat sends a message to a specific assistant and streams the response.
func (c *Client) AssistantChat(ctx context.Context, assistant, message string, callbacks AssistantChatCallbacks) (*AskResponse, error) {
	reqBody, err := json.Marshal(AssistantChatRequest{Message: message})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/assistants/%s/chat", c.baseURL, assistant)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
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

	return c.readAssistantChatStream(ctx, resp, callbacks)
}

func (c *Client) readAssistantChatStream(ctx context.Context, resp *http.Response, callbacks AssistantChatCallbacks) (*AskResponse, error) {
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
			case "assistant":
				var info AssistantInfo
				if err := json.Unmarshal([]byte(data), &info); err == nil {
					if callbacks.OnAssistant != nil {
						callbacks.OnAssistant(info)
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
				}
			}

			currentEvent = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return &AskResponse{Message: fullContent.String()}, err
	}

	if result.Message == "" {
		result.Message = fullContent.String()
	}

	return &result, nil
}
