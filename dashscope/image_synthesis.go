/*
 * @Author: Ceoifung
 * @Date: 2026-02-10 18:18:45
 * @LastEditors: Ceoifung
 * @LastEditTime: 2026-02-10 19:17:38
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

// Image Synthesis Models
const (
	WanxV1              = "wanx-v1"
	WanxSketchToImageV1 = "wanx-sketch-to-image-v1"
	WanxBackgroundV1    = "wanx-background-generation-v2"
)

// ImageSynthesis handles the image synthesis API.
type ImageSynthesis struct {
	APIKey string
	client *http.Client
}

// NewImageSynthesis creates a new ImageSynthesis client.
func NewImageSynthesis(apiKey string) *ImageSynthesis {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &ImageSynthesis{
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// SetHTTPClient sets a custom HTTP client.
func (s *ImageSynthesis) SetHTTPClient(client *http.Client) {
	s.client = client
}

type ImageSynthesisRequest struct {
	Model      string                    `json:"model"`
	Input      ImageSynthesisInput       `json:"input"`
	Parameters *ImageSynthesisParameters `json:"parameters,omitempty"`
}

type ImageSynthesisInput struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	RefImg         string `json:"ref_img,omitempty"`
	SketchImageURL string `json:"sketch_image_url,omitempty"`
	BaseImageURL   string `json:"base_image_url,omitempty"` // For background generation
}

type ImageSynthesisParameters struct {
	N             int     `json:"n,omitempty"`
	Size          string  `json:"size,omitempty"`
	Style         string  `json:"style,omitempty"` // "<auto>" or specific style
	Seed          int     `json:"seed,omitempty"`
	Similarity    float64 `json:"similarity,omitempty"`    // for sketch
	SketchWeight  int     `json:"sketch_weight,omitempty"` // for sketch
	Realisticness int     `json:"realisticness,omitempty"` // for sketch
}

type ImageSynthesisResult struct {
	URL string `json:"url"`
}

type ImageSynthesisResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID     string                 `json:"task_id"`
		TaskStatus string                 `json:"task_status"`
		Results    []ImageSynthesisResult `json:"results"`
	} `json:"output"`
	Usage      ImageSynthesisUsage `json:"usage"`
	StatusCode int                 `json:"status_code,omitempty"`
	Message    string              `json:"message,omitempty"`
}

type ImageSynthesisUsage struct {
	ImageCount int `json:"image_count"`
}

// Call performs the image synthesis request (submit and wait).
func (s *ImageSynthesis) Call(ctx context.Context, req ImageSynthesisRequest) (*ImageSynthesisResponse, error) {
	// 1. Submit async task
	taskID, err := s.AsyncCall(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. Wait for task completion
	// Note: WaitForTask needs to be updated to support Context and potentially Client.
	// For now, we'll keep using the standalone function but we should update it.
	// Actually, let's update WaitForTask to accept context later.
	taskResp, err := WaitForTask(ctx, s.APIKey, taskID)
	if err != nil {
		return nil, err
	}

	// 3. Convert generic TaskResponse to ImageSynthesisResponse
	resp := &ImageSynthesisResponse{
		RequestID:  taskResp.RequestID,
		StatusCode: taskResp.StatusCode,
		Message:    taskResp.Message,
	}
	resp.Output.TaskID = taskResp.Output.TaskID
	resp.Output.TaskStatus = taskResp.Output.TaskStatus

	if len(taskResp.Output.Results) > 0 {
		if err := json.Unmarshal(taskResp.Output.Results, &resp.Output.Results); err != nil {
			return nil, fmt.Errorf("failed to unmarshal results: %v", err)
		}
	}

	// Usage parsing if available (TaskResponse.Usage is interface{})
	if taskResp.Usage != nil {
		usageBytes, _ := json.Marshal(taskResp.Usage)
		json.Unmarshal(usageBytes, &resp.Usage)
	}

	return resp, nil
}

// AsyncCall submits the image synthesis task and returns the task ID.
func (s *ImageSynthesis) AsyncCall(ctx context.Context, req ImageSynthesisRequest) (string, error) {
	url := ImageSynthesisURL

	// Basic routing based on known models/tasks if needed,
	// but standard text2image uses the above URL.
	// For background generation, it might be different, but let's stick to text2image for now.

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-DashScope-Async", "enable")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("async call failed: %s (%s)", taskResp.Message, taskResp.Code)
	}

	return taskResp.Output.TaskID, nil
}
