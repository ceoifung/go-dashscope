package dashscope

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TaskStatus constants
const (
	TaskStatusPending   = "PENDING"
	TaskStatusRunning   = "RUNNING"
	TaskStatusSucceeded = "SUCCEEDED"
	TaskStatusFailed    = "FAILED"
	TaskStatusCanceled  = "CANCELED"
	TaskStatusUnknown   = "UNKNOWN"
)

// TaskResponse represents the generic response structure for async tasks.
type TaskResponse struct {
	RequestID  string      `json:"request_id"`
	StatusCode int         `json:"status_code,omitempty"`
	Code       string      `json:"code,omitempty"`
	Message    string      `json:"message,omitempty"`
	Output     TaskOutput  `json:"output"`
	Usage      interface{} `json:"usage,omitempty"` // Usage can vary
}

// TaskOutput represents the common output fields for async tasks.
type TaskOutput struct {
	TaskID     string          `json:"task_id"`
	TaskStatus string          `json:"task_status"`
	Results    json.RawMessage `json:"results,omitempty"` // Keep raw JSON for specific parsing
}

// GetTask retrieves the status of an asynchronous task.
func GetTask(ctx context.Context, apiKey string, taskID string) (*TaskResponse, error) {
	return GetTaskWithClient(ctx, apiKey, taskID, nil)
}

// GetTaskWithClient retrieves the status of an asynchronous task using a custom HTTP client.
func GetTaskWithClient(ctx context.Context, apiKey string, taskID string, client *http.Client) (*TaskResponse, error) {
	url := fmt.Sprintf("%s/%s", TaskBaseURL, taskID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &taskResp, fmt.Errorf("get task failed: %s (%s)", taskResp.Message, taskResp.Code)
	}

	return &taskResp, nil
}

// WaitForTask waits for an asynchronous task to complete.
func WaitForTask(ctx context.Context, apiKey string, taskID string) (*TaskResponse, error) {
	return WaitForTaskWithClient(ctx, apiKey, taskID, nil)
}

// WaitForTaskWithClient waits for an asynchronous task to complete using a custom HTTP client.
func WaitForTaskWithClient(ctx context.Context, apiKey string, taskID string, client *http.Client) (*TaskResponse, error) {
	waitSeconds := 1 * time.Second
	maxWaitSeconds := 5 * time.Second
	incrementSteps := 3
	step := 0

	for {
		step++
		resp, err := GetTaskWithClient(ctx, apiKey, taskID, client)
		if err != nil {
			// Network error or other immediate failure
			return nil, err
		}

		if resp.Output.TaskStatus == TaskStatusSucceeded ||
			resp.Output.TaskStatus == TaskStatusFailed ||
			resp.Output.TaskStatus == TaskStatusCanceled {
			return resp, nil
		}

		// Wait logic similar to Python SDK
		if waitSeconds < maxWaitSeconds && step%incrementSteps == 0 {
			waitSeconds *= 2
			if waitSeconds > maxWaitSeconds {
				waitSeconds = maxWaitSeconds
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(waitSeconds):
			// continue
		}
	}
}
