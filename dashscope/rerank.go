package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ReRank Models
const (
	GteRerank = "gte-rerank"
)

// TextReRank handles the text rerank API.
type TextReRank struct {
	APIKey string
	client *http.Client
}

// NewTextReRank creates a new TextReRank client.
func NewTextReRank(apiKey string) *TextReRank {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &TextReRank{
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// SetHTTPClient sets a custom HTTP client.
func (r *TextReRank) SetHTTPClient(client *http.Client) {
	r.client = client
}

type TextReRankRequest struct {
	Model      string                `json:"model"`
	Input      TextReRankInput       `json:"input"`
	Parameters *TextReRankParameters `json:"parameters,omitempty"`
}

type TextReRankInput struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type TextReRankParameters struct {
	ReturnDocuments bool `json:"return_documents,omitempty"`
	TopN            int  `json:"top_n,omitempty"`
}

type TextReRankResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Results []ReRankResult `json:"results"`
	} `json:"output"`
	Usage      ReRankUsage `json:"usage"`
	StatusCode int         `json:"status_code,omitempty"`
	Message    string      `json:"message,omitempty"`
}

type ReRankResult struct {
	Index          int         `json:"index"`
	RelevanceScore float64     `json:"relevance_score"`
	Document       interface{} `json:"document,omitempty"` // Map or Struct depending on return_documents
}

type ReRankUsage struct {
	TotalTokens int `json:"total_tokens"`
}

// Call performs the text rerank request.
func (r *TextReRank) Call(ctx context.Context, req TextReRankRequest) (*TextReRankResponse, error) {
	url := TextReRankURL

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+r.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rrResp TextReRankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rrResp); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &rrResp, fmt.Errorf("text rerank failed: %s (%s)", rrResp.Message, rrResp.RequestID)
	}

	return &rrResp, nil
}
