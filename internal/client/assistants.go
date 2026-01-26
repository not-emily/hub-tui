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
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Persona     string              `json:"persona"`
	Modules     []string            `json:"modules,omitempty"`
	Gather      map[string][]string `json:"gather,omitempty"`
	LLMProfile  string              `json:"llm_profile"`
	Keywords    []string            `json:"keywords,omitempty"`
	Enabled     bool                `json:"enabled"`
}

// CreateAssistantRequest is the request body for creating an assistant.
type CreateAssistantRequest struct {
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Persona     string              `json:"persona"`
	Modules     []string            `json:"modules,omitempty"`
	Gather      map[string][]string `json:"gather,omitempty"`
	LLMProfile  string              `json:"llm_profile"`
	Keywords    []string            `json:"keywords,omitempty"`
}

// UpdateAssistantRequest is the request body for updating an assistant.
type UpdateAssistantRequest struct {
	DisplayName string              `json:"display_name,omitempty"`
	Persona     string              `json:"persona,omitempty"`
	Modules     []string            `json:"modules,omitempty"`
	Gather      map[string][]string `json:"gather,omitempty"`
	LLMProfile  string              `json:"llm_profile,omitempty"`
	Keywords    []string            `json:"keywords,omitempty"`
}

// AssistantMemory represents an assistant's core memory.
type AssistantMemory struct {
	Entries map[string]string `json:"entries"`
}

// AssistantTemplate represents an assistant template from a module.
type AssistantTemplate struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Persona     string   `json:"persona"`
	Modules     []string `json:"modules,omitempty"`
	LLMProfile  string   `json:"llm_profile"`
	Keywords    []string `json:"keywords,omitempty"`
}

// TemplateOverrides allows overriding template fields when creating from template.
type TemplateOverrides struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// assistantsResponse is the API response wrapper.
type assistantsResponse struct {
	Assistants []Assistant `json:"assistants"`
}

// assistantResponse is the API response wrapper for a single assistant.
type assistantResponse struct {
	Assistant Assistant `json:"assistant"`
}

// templatesResponse is the API response wrapper for assistant templates.
type templatesResponse struct {
	Module    string              `json:"module"`
	Templates []AssistantTemplate `json:"templates"`
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

// GetAssistant fetches a single assistant by name.
func (c *Client) GetAssistant(name string) (*Assistant, error) {
	resp, err := c.get("/assistants/" + name)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	// API returns assistant directly (not wrapped)
	var assistant Assistant
	if err := json.NewDecoder(resp.Body).Decode(&assistant); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &assistant, nil
}

// CreateAssistant creates a new assistant.
func (c *Client) CreateAssistant(req *CreateAssistantRequest) (*Assistant, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.post("/assistants", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, parseError(resp)
	}

	// Re-fetch the assistant to get the full object
	return c.GetAssistant(req.Name)
}

// UpdateAssistant updates an existing assistant.
func (c *Client) UpdateAssistant(name string, req *UpdateAssistantRequest) (*Assistant, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.put("/assistants/"+name, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	// Re-fetch the assistant to get the full object
	return c.GetAssistant(name)
}

// DeleteAssistant deletes an assistant.
func (c *Client) DeleteAssistant(name string) error {
	resp, err := c.delete("/assistants/" + name)
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return parseError(resp)
	}

	return nil
}

// GetAssistantMemory fetches an assistant's core memory.
func (c *Client) GetAssistantMemory(name string) (*AssistantMemory, error) {
	resp, err := c.get("/assistants/" + name + "/memory")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var memory AssistantMemory
	if err := json.NewDecoder(resp.Body).Decode(&memory); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return &memory, nil
}

// UpdateAssistantMemory updates an assistant's core memory.
func (c *Client) UpdateAssistantMemory(name string, memory *AssistantMemory) error {
	body, err := json.Marshal(memory)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.put("/assistants/"+name+"/memory", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}

	return nil
}

// ClearAssistantHistory clears an assistant's conversation history.
func (c *Client) ClearAssistantHistory(name string) error {
	resp, err := c.delete("/assistants/" + name + "/history")
	if err != nil {
		return fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return parseError(resp)
	}

	return nil
}

// ListModuleAssistantTemplates fetches assistant templates for a module.
func (c *Client) ListModuleAssistantTemplates(module string) ([]AssistantTemplate, error) {
	resp, err := c.get("/modules/" + module + "/assistants")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}

	var result templatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}

	return result.Templates, nil
}

// CreateAssistantFromTemplate creates an assistant from a module template.
func (c *Client) CreateAssistantFromTemplate(module, template string, overrides *TemplateOverrides) (*Assistant, error) {
	var body []byte
	var err error

	if overrides != nil {
		body, err = json.Marshal(overrides)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	} else {
		body = []byte("{}")
	}

	resp, err := c.post("/modules/"+module+"/assistants/"+template+"/create", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, parseError(resp)
	}

	// Try to get the assistant name from response, then fetch full object
	var result struct {
		Assistant string `json:"assistant"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If we can't parse the name, use the template name with overrides
		name := template
		if overrides != nil && overrides.Name != "" {
			name = overrides.Name
		}
		return c.GetAssistant(name)
	}

	return c.GetAssistant(result.Assistant)
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
