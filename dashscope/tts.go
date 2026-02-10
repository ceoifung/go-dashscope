package dashscope

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// SpeechSynthesisUsage represents the usage statistics.
type SpeechSynthesisUsage struct {
	Characters int `json:"characters"`
}

// SpeechSynthesisResult represents the result of a speech synthesis event.
type SpeechSynthesisResult struct {
	AudioFrame []byte                 // Audio data for this frame
	AudioData  []byte                 // Complete audio data (only available in final result)
	Sentence   map[string]interface{} // Sentence level timestamp info
	Sentences  []interface{}          // Complete timestamp info (only available in final result)
	Response   map[string]interface{} // Full response payload
	Usage      *SpeechSynthesisUsage  // Usage statistics (only available in final result)
}

// Save saves the accumulated audio data to a file.
func (r *SpeechSynthesisResult) Save(filename string) error {
	if len(r.AudioData) == 0 {
		return errors.New("no audio data to save")
	}
	return os.WriteFile(filename, r.AudioData, 0644)
}

// ResultCallback defines the interface for handling synthesis events.
type ResultCallback interface {
	OnOpen()
	OnComplete()
	OnError(err error)
	OnClose()
	OnEvent(result *SpeechSynthesisResult)
}

// DefaultCallback is a helper struct that implements ResultCallback with empty methods.
// Users can embed this struct and override only the methods they need.
type DefaultCallback struct{}

func (d *DefaultCallback) OnOpen()                               {}
func (d *DefaultCallback) OnComplete()                           {}
func (d *DefaultCallback) OnError(err error)                     {}
func (d *DefaultCallback) OnClose()                              {}
func (d *DefaultCallback) OnEvent(result *SpeechSynthesisResult) {}

// SpeechSynthesizer is the client for text-to-speech.
type SpeechSynthesizer struct {
	Model     string
	APIKey    string
	Workspace string
}

// NewSpeechSynthesizer creates a new synthesizer.
func NewSpeechSynthesizer(model, apiKey string) *SpeechSynthesizer {
	return &SpeechSynthesizer{
		Model:  model,
		APIKey: apiKey,
	}
}

// SetWorkspace sets the workspace ID.
func (s *SpeechSynthesizer) SetWorkspace(workspace string) {
	s.Workspace = workspace
}

type wsRequestHeader struct {
	Action    string `json:"action"`
	TaskID    string `json:"task_id"`
	Streaming string `json:"streaming"`
}

type wsRequestPayload struct {
	Model      string                 `json:"model"`
	TaskGroup  string                 `json:"task_group"`
	Task       string                 `json:"task"`
	Function   string                 `json:"function"`
	Input      map[string]interface{} `json:"input"`
	Parameters map[string]interface{} `json:"parameters"`
}

type wsRequest struct {
	Header  wsRequestHeader  `json:"header"`
	Payload wsRequestPayload `json:"payload"`
}

type wsResponseHeader struct {
	Event   string `json:"event"`
	TaskID  string `json:"task_id"`
	Code    string `json:"error_code,omitempty"`
	Message string `json:"error_message,omitempty"`
}

type wsResponse struct {
	Header  wsResponseHeader       `json:"header"`
	Payload map[string]interface{} `json:"payload"`
}

// Call performs the text-to-speech synthesis.
// parameters can include: format, sample_rate, volume, rate, pitch, etc.
// Returns the final result containing all audio data and sentences.
func (s *SpeechSynthesizer) Call(ctx context.Context, text string, callback ResultCallback, parameters map[string]interface{}) (*SpeechSynthesisResult, error) {
	if s.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	// 1. Prepare WebSocket connection
	header := http.Header{}
	header.Set("Authorization", "bearer "+s.APIKey)
	header.Set("User-Agent", "dashscope-go-sdk/0.1.0")
	if s.Workspace != "" {
		header.Set("X-DashScope-WorkSpace", s.Workspace)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, BaseWebsocketURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to websocket: %w", err)
	}
	defer conn.Close()

	if callback != nil {
		callback.OnOpen()
		defer callback.OnClose()
	}

	// 2. Send Start Task Request
	taskID := strings.ReplaceAll(uuid.New().String(), "-", "")
	req := wsRequest{
		Header: wsRequestHeader{
			Action:    string(ActionRunTask),
			TaskID:    taskID,
			Streaming: "out",
		},
		Payload: wsRequestPayload{
			Model:     s.Model,
			TaskGroup: TaskGroupAudio,
			Task:      TaskTTS,
			Function:  FunctionTTS,
			Input: map[string]interface{}{
				"text": text,
			},
			Parameters: parameters,
		},
	}

	if err := conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	finalResult := &SpeechSynthesisResult{
		AudioData: make([]byte, 0),
		Sentences: make([]interface{}, 0),
	}

	// 3. Receive Loop
	for {
		messageType, messageData, err := conn.ReadMessage()
		if err != nil {
			if callback != nil {
				callback.OnError(err)
			}
			return nil, err
		}

		if messageType == websocket.BinaryMessage {
			// Audio data
			finalResult.AudioData = append(finalResult.AudioData, messageData...)

			if callback != nil {
				callback.OnEvent(&SpeechSynthesisResult{
					AudioFrame: messageData,
				})
			}
		} else if messageType == websocket.TextMessage {
			var resp wsResponse
			if err := json.Unmarshal(messageData, &resp); err != nil {
				if callback != nil {
					callback.OnError(fmt.Errorf("failed to unmarshal response: %w", err))
				}
				continue
			}

			switch EventType(resp.Header.Event) {
			case EventTaskStarted:
				// Task started, waiting for generation
			case EventResultGenerated, EventTaskFinished:
				// Check for usage info
				if usageVal, ok := resp.Payload["usage"]; ok {
					if usageMap, ok := usageVal.(map[string]interface{}); ok {
						chars, _ := usageMap["characters"].(float64)
						finalResult.Usage = &SpeechSynthesisUsage{
							Characters: int(chars),
						}
					}
				}

				if EventType(resp.Header.Event) == EventTaskFinished {
					if callback != nil {
						callback.OnComplete()
					}
					return finalResult, nil
				}

				// Check if there is sentence info
				if sentence, ok := resp.Payload["sentence"]; ok {
					finalResult.Sentences = append(finalResult.Sentences, sentence)

					if callback != nil {
						sMap, _ := sentence.(map[string]interface{})
						callback.OnEvent(&SpeechSynthesisResult{
							Sentence: sMap,
							Response: resp.Payload,
						})
					}
				} else {
					// Just generic response or meta info (like usage only)
					if callback != nil {
						callback.OnEvent(&SpeechSynthesisResult{
							Response: resp.Payload,
						})
					}
				}
			case EventTaskFailed:
				err := fmt.Errorf("task failed: %s - %s", resp.Header.Code, resp.Header.Message)
				if callback != nil {
					callback.OnError(err)
				}
				return nil, err
			default:
				// Unknown event
			}
		}
	}
}
