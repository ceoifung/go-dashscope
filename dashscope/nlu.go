/*
 * @Author: Ceoifung
 * @Date: 2026-02-10 18:24:50
 * @LastEditors: Ceoifung
 * @LastEditTime: 2026-02-10 19:18:31
 * @Description: XiaoRGEEK All Rights Reserved. Powered By Ceoifung
 */
package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// NLU Models
const (
	OpenNLUV1 = "opennlu-v1"
)

// Understanding handles the NLU API.
type Understanding struct {
	APIKey string
	client *http.Client
}

// NewUnderstanding creates a new Understanding client.
func NewUnderstanding(apiKey string) *Understanding {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &Understanding{
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// SetHTTPClient sets a custom HTTP client.
func (u *Understanding) SetHTTPClient(client *http.Client) {
	u.client = client
}

type UnderstandingRequest struct {
	Model      string                   `json:"model"`
	Input      UnderstandingInput       `json:"input"`
	Parameters *UnderstandingParameters `json:"parameters,omitempty"`
}

type UnderstandingInput struct {
	Sentence string `json:"sentence"`
	Labels   string `json:"labels"`         // Comma separated
	Task     string `json:"task,omitempty"` // "extraction" or "classification"
}

type UnderstandingParameters struct {
	// Add parameters if needed, empty for now based on python SDK
}

type UnderstandingResponse struct {
	RequestID  string             `json:"request_id"`
	Output     json.RawMessage    `json:"output"` // Use RawMessage to inspect structure
	Usage      UnderstandingUsage `json:"usage"`
	StatusCode int                `json:"status_code,omitempty"`
	Message    string             `json:"message,omitempty"`
}

type UnderstandingUsage struct {
	TotalTokens int `json:"total_tokens"`
}

// Call performs the understanding request.
func (u *Understanding) Call(ctx context.Context, req UnderstandingRequest) (*UnderstandingResponse, error) {
	url := NLUUnderstandingURL

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+u.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var uResp UnderstandingResponse
	if err := json.NewDecoder(resp.Body).Decode(&uResp); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &uResp, fmt.Errorf("understanding failed: %s (%s)", uResp.Message, uResp.RequestID)
	}

	return &uResp, nil
}
