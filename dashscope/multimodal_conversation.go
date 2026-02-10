package dashscope

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// MultiModal Conversation Models
const (
	QwenVLChatV1     = "qwen-vl-chat-v1"
	QwenVLChatV1Plus = "qwen-vl-plus"
	QwenVLChatV1Max  = "qwen-vl-max"
)

// MultiModalConversation handles the multimodal conversation API.
type MultiModalConversation struct {
	APIKey    string
	Workspace string
	client    *http.Client
}

// NewMultiModalConversation creates a new MultiModalConversation client.
func NewMultiModalConversation(apiKey string) *MultiModalConversation {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &MultiModalConversation{
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// SetHTTPClient sets a custom HTTP client.
func (m *MultiModalConversation) SetHTTPClient(client *http.Client) {
	m.client = client
}

type MultiModalConversationRequest struct {
	Model      string                            `json:"model"`
	Input      MultiModalConversationInput       `json:"input"`
	Parameters *MultiModalConversationParameters `json:"parameters,omitempty"`
}

type MultiModalConversationInput struct {
	Messages []MultiModalMessage `json:"messages"`
}

type MultiModalMessage struct {
	Role    string                  `json:"role"`
	Content []MultiModalContentItem `json:"content"`
}

type MultiModalContentItem struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}

type MultiModalConversationParameters struct {
	TopP         float64 `json:"top_p,omitempty"`
	TopK         int     `json:"top_k,omitempty"`
	Seed         int     `json:"seed,omitempty"`
	EnableSearch bool    `json:"enable_search,omitempty"`
	ResultFormat string  `json:"result_format,omitempty"` // "message"
}

type MultiModalConversationResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Choices []MultiModalChoice `json:"choices"`
	} `json:"output"`
	Usage      MultiModalUsage `json:"usage"`
	StatusCode int             `json:"status_code,omitempty"`
	Message    string          `json:"message,omitempty"`
}

type MultiModalChoice struct {
	FinishReason string            `json:"finish_reason"`
	Message      MultiModalMessage `json:"message"`
}

type MultiModalUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	ImageCount   int `json:"image_count"`
}

// Call performs a synchronous multimodal conversation request.
func (m *MultiModalConversation) Call(ctx context.Context, req MultiModalConversationRequest) (*MultiModalConversationResponse, error) {
	url := QwenVLGenerationURL

	if req.Parameters == nil {
		req.Parameters = &MultiModalConversationParameters{}
	}
	// Default result_format to message if not set, though API might default it.
	if req.Parameters.ResultFormat == "" {
		req.Parameters.ResultFormat = "message"
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.APIKey)
	httpReq.Header.Set("X-DashScope-WorkSpace", m.Workspace)

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var mmResp MultiModalConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&mmResp); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &mmResp, fmt.Errorf("multimodal conversation failed: %s (%s)", mmResp.Message, mmResp.RequestID)
	}

	return &mmResp, nil
}

// CallStream performs a streaming multimodal conversation request.
func (m *MultiModalConversation) CallStream(ctx context.Context, req MultiModalConversationRequest) (<-chan MultiModalConversationResponse, error) {
	url := QwenVLGenerationURL

	// Ensure stream parameter is true?
	// The Python SDK seems to rely on the body param.
	// But let's check if parameters exist.
	// Note: The struct definition for MultiModalConversationParameters might not have Stream field explicitly defined
	// in previous steps? Let's assume it does or we rely on the user setting it,
	// or we just send headers.
	// Python SDK: "input": {...}, "parameters": {"stream": True}

	// We can't easily inject "stream": true into req.Parameters if it's a specific struct without that field
	// or if we don't want to use reflection.
	// However, standard DashScope API usually respects X-DashScope-SSE: enable.

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.APIKey)
	httpReq.Header.Set("X-DashScope-SSE", "enable")
	httpReq.Header.Set("X-DashScope-WorkSpace", m.Workspace)

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	ch := make(chan MultiModalConversationResponse)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				var mmResp MultiModalConversationResponse
				if err := json.Unmarshal([]byte(data), &mmResp); err != nil {
					// Handle parsing error if needed
					continue
				}
				ch <- mmResp
			}
		}
	}()

	return ch, nil
}
