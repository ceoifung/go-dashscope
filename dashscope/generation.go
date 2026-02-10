package dashscope

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Generation models
const (
	QwenTurbo = "qwen-turbo"
	QwenPlus  = "qwen-plus"
	QwenMax   = "qwen-max"
	QwenFlash = "qwen-flash"
)

// GenerationRequest represents the request body for generation.
type GenerationRequest struct {
	Model      string                `json:"model"`
	Input      GenerationInput       `json:"input"`
	Parameters *GenerationParameters `json:"parameters,omitempty"`
}

// GenerationInput represents the input for generation.
type GenerationInput struct {
	Prompt   string    `json:"prompt,omitempty"`
	Messages []Message `json:"messages,omitempty"`
}

// GenerationParameters represents the parameters for generation.
type GenerationParameters struct {
	ResultFormat      string      `json:"result_format,omitempty"`
	Seed              uint64      `json:"seed,omitempty"`
	MaxTokens         int         `json:"max_tokens,omitempty"`
	TopP              float64     `json:"top_p,omitempty"`
	TopK              int         `json:"top_k,omitempty"`
	RepetitionPenalty float64     `json:"repetition_penalty,omitempty"`
	Temperature       float64     `json:"temperature,omitempty"`
	Stop              interface{} `json:"stop,omitempty"` // string or []string
	EnableSearch      bool        `json:"enable_search,omitempty"`
	IncrementalOutput bool        `json:"incremental_output,omitempty"`
	Stream            bool        `json:"stream,omitempty"`
}

// GenerationResponse represents the response from generation.
type GenerationResponse struct {
	RequestID  string           `json:"request_id"`
	Output     GenerationOutput `json:"output"`
	Usage      GenerationUsage  `json:"usage"`
	StatusCode int              `json:"status_code,omitempty"`
	Code       string           `json:"code,omitempty"`
	Message    string           `json:"message,omitempty"`
}

// GenerationOutput represents the output data in the response.
type GenerationOutput struct {
	Text         string   `json:"text,omitempty"`
	FinishReason string   `json:"finish_reason,omitempty"`
	Choices      []Choice `json:"choices,omitempty"`
}

// Choice represents a choice in the output.
type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

// GenerationUsage represents the token usage.
type GenerationUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Generation handles the text generation API.
type Generation struct {
	APIKey    string
	Workspace string
	client    *http.Client
}

// NewGeneration creates a new Generation client.
func NewGeneration(apiKey string) *Generation {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &Generation{
		APIKey: apiKey,
		client: &http.Client{}, // Default client with no timeout
	}
}

// SetHTTPClient sets a custom HTTP client.
func (g *Generation) SetHTTPClient(client *http.Client) {
	g.client = client
}

// SetWorkspace sets the workspace ID.
func (g *Generation) SetWorkspace(workspace string) {
	g.Workspace = workspace
}

// Call performs a synchronous generation request.
func (g *Generation) Call(ctx context.Context, req GenerationRequest) (*GenerationResponse, error) {
	url := QwenGenerationURL

	if req.Parameters == nil {
		req.Parameters = &GenerationParameters{}
	}
	req.Parameters.Stream = false

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
	if g.Workspace != "" {
		httpReq.Header.Set("X-DashScope-WorkSpace", g.Workspace)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result GenerationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	result.StatusCode = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("API error: %s (code: %s, request_id: %s)", result.Message, result.Code, result.RequestID)
	}

	return &result, nil
}

// CallStream performs a streaming generation request.
// It returns a channel that receives GenerationResponse updates.
func (g *Generation) CallStream(ctx context.Context, req GenerationRequest) (<-chan GenerationResponse, error) {
	url := QwenGenerationURL

	if req.Parameters == nil {
		req.Parameters = &GenerationParameters{}
	}
	req.Parameters.Stream = true
	// Enable incremental output by default for streaming if not specified
	// if !req.Parameters.IncrementalOutput {
	// 	req.Parameters.IncrementalOutput = true
	// }

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("X-DashScope-SSE", "enable") // Python SDK sends this via 'stream' param in body, but standard SSE might need headers?
	// Wait, Python SDK sends stream=True in body.
	// It relies on 'stream' parameter in the JSON body, not Accept header primarily, but Requests lib might handle it?
	// api_request_factory.py doesn't seem to set Accept: text/event-stream explicitly for HTTP,
	// but maybe the server responds with it if stream=True in body.

	if g.Workspace != "" {
		httpReq.Header.Set("X-DashScope-WorkSpace", g.Workspace)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	ch := make(chan GenerationResponse)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				var result GenerationResponse
				if err := json.Unmarshal([]byte(data), &result); err != nil {
					// Handle error or log?
					continue
				}
				result.StatusCode = resp.StatusCode
				ch <- result
			}
		}
	}()

	return ch, nil
}
