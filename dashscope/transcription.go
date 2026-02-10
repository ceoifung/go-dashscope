package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Transcription Models
const (
	ParaformerV1      = "paraformer-v1"
	Paraformer8kV1    = "paraformer-8k-v1"
	ParaformerMtlV1   = "paraformer-mtl-v1"
	ParaformerRealtimeV1 = "paraformer-realtime-v1" // For recognition, but good to have constants
)

// Transcription handles the audio transcription API.
type Transcription struct {
	APIKey string
	client *http.Client
}

// NewTranscription creates a new Transcription client.
func NewTranscription(apiKey string) *Transcription {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &Transcription{
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// SetHTTPClient sets a custom HTTP client.
func (t *Transcription) SetHTTPClient(client *http.Client) {
	t.client = client
}

type TranscriptionRequest struct {
	Model      string                  `json:"model"`
	Input      TranscriptionInput      `json:"input"`
	Parameters *TranscriptionParameters `json:"parameters,omitempty"`
	Resources  []Resource              `json:"resources,omitempty"`
}

type TranscriptionInput struct {
	FileURLs []string `json:"file_urls"`
}

type Resource struct {
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
}

type TranscriptionParameters struct {
	ChannelID                  []int  `json:"channel_id,omitempty"`
	DisfluencyRemovalEnabled   *bool  `json:"disfluency_removal_enabled,omitempty"`
	DiarizationEnabled         *bool  `json:"diarization_enabled,omitempty"`
	SpeakerCount               int    `json:"speaker_count,omitempty"`
	TimestampAlignmentEnabled  *bool  `json:"timestamp_alignment_enabled,omitempty"`
	SpecialWordFilter          string `json:"special_word_filter,omitempty"`
	AudioEventDetectionEnabled *bool  `json:"audio_event_detection_enabled,omitempty"`
}

type TranscriptionResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID     string          `json:"task_id"`
		TaskStatus string          `json:"task_status"`
		Results    json.RawMessage `json:"results,omitempty"` // Results structure varies
	} `json:"output"`
	Usage      json.RawMessage `json:"usage,omitempty"`
	StatusCode int             `json:"status_code,omitempty"`
	Message    string          `json:"message,omitempty"`
}

// Call performs the transcription request (submit and wait).
func (t *Transcription) Call(ctx context.Context, req TranscriptionRequest) (*TranscriptionResponse, error) {
	// 1. Submit async task
	taskID, err := t.AsyncCall(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. Wait for task completion
	taskResp, err := WaitForTask(ctx, t.APIKey, taskID)
	if err != nil {
		return nil, err
	}

	// 3. Convert generic TaskResponse to TranscriptionResponse
	resp := &TranscriptionResponse{
		RequestID:  taskResp.RequestID,
		StatusCode: taskResp.StatusCode,
		Message:    taskResp.Message,
		Usage:      json.RawMessage(nil), // Handle usage if needed
	}
	resp.Output.TaskID = taskResp.Output.TaskID
	resp.Output.TaskStatus = taskResp.Output.TaskStatus
	resp.Output.Results = taskResp.Output.Results
	
	if taskResp.Usage != nil {
		usageBytes, _ := json.Marshal(taskResp.Usage)
		resp.Usage = usageBytes
	}

	return resp, nil
}

// AsyncCall submits the transcription task and returns the task ID.
func (t *Transcription) AsyncCall(ctx context.Context, req TranscriptionRequest) (string, error) {
	url := ASRTranscriptionURL

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Authorization", "Bearer "+t.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
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
