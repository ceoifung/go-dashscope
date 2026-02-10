package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Text Embedding Models
const (
	TextEmbeddingV1 = "text-embedding-v1"
	TextEmbeddingV2 = "text-embedding-v2"
	TextEmbeddingV3 = "text-embedding-v3"
	TextEmbeddingV4 = "text-embedding-v4" // if available, strictly following python SDK constants
)

// Multimodal Embedding Models
const (
	MultimodalEmbeddingOnePeaceV1 = "multimodal-embedding-one-peace-v1"
)

// TextEmbedding handles the text embedding API.
type TextEmbedding struct {
	APIKey string
	client *http.Client
}

// NewTextEmbedding creates a new TextEmbedding client.
func NewTextEmbedding(apiKey string) *TextEmbedding {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &TextEmbedding{
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// SetHTTPClient sets a custom HTTP client.
func (e *TextEmbedding) SetHTTPClient(client *http.Client) {
	e.client = client
}

type TextEmbeddingRequest struct {
	Model      string                   `json:"model"`
	Input      TextEmbeddingInput       `json:"input"`
	Parameters *TextEmbeddingParameters `json:"parameters,omitempty"`
}

type TextEmbeddingInput struct {
	Texts []string `json:"texts"`
}

type TextEmbeddingParameters struct {
	TextType string `json:"text_type,omitempty"` // "query" or "document"
}

type TextEmbeddingResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Embeddings []EmbeddingResult `json:"embeddings"`
	} `json:"output"`
	Usage      EmbeddingUsage `json:"usage"`
	StatusCode int            `json:"status_code,omitempty"`
	Message    string         `json:"message,omitempty"`
}

type EmbeddingResult struct {
	TextIndex int       `json:"text_index"`
	Embedding []float64 `json:"embedding"`
}

type EmbeddingUsage struct {
	TotalTokens int `json:"total_tokens"`
}

// Call performs the text embedding request.
func (e *TextEmbedding) Call(ctx context.Context, req TextEmbeddingRequest) (*TextEmbeddingResponse, error) {
	url := TextEmbeddingURL

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+e.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var embeddingResp TextEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &embeddingResp, fmt.Errorf("text embedding failed: %s (%s)", embeddingResp.Message, embeddingResp.RequestID)
	}

	return &embeddingResp, nil
}

// MultimodalEmbedding handles the multimodal embedding API.
type MultimodalEmbedding struct {
	APIKey string
}

// NewMultimodalEmbedding creates a new MultimodalEmbedding client.
func NewMultimodalEmbedding(apiKey string) *MultimodalEmbedding {
	return &MultimodalEmbedding{APIKey: apiKey}
}

type MultimodalEmbeddingRequest struct {
	Model      string                         `json:"model"`
	Input      MultimodalEmbeddingInput       `json:"input"`
	Parameters *MultimodalEmbeddingParameters `json:"parameters,omitempty"`
}

type MultimodalEmbeddingInput struct {
	Contents []MultimodalContent `json:"contents"`
}

type MultimodalContent struct {
	Text   string  `json:"text,omitempty"`
	Image  string  `json:"image,omitempty"`
	Audio  string  `json:"audio,omitempty"`
	Factor float64 `json:"factor"`
}

type MultimodalEmbeddingParameters struct {
	AutoTruncation bool `json:"auto_truncation,omitempty"`
}

type MultimodalEmbeddingResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Embedding []float64 `json:"embedding"`
	} `json:"output"`
	Usage      EmbeddingUsage `json:"usage"`
	StatusCode int            `json:"status_code,omitempty"`
	Message    string         `json:"message,omitempty"`
}

// Call performs the multimodal embedding request.
func (e *MultimodalEmbedding) Call(req MultimodalEmbeddingRequest) (*MultimodalEmbeddingResponse, error) {
	url := "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding"

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+e.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	// Header required for OSS resource resolution if URLs are passed (assumed true for simplicity)
	httpReq.Header.Set("X-DashScope-OssResourceResolve", "enable")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var embeddingResp MultimodalEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &embeddingResp, fmt.Errorf("multimodal embedding failed: %s (%s)", embeddingResp.Message, embeddingResp.RequestID)
	}

	return &embeddingResp, nil
}
