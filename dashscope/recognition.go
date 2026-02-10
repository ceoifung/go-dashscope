package dashscope

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// RecognitionCallback defines the interface for handling recognition events.
type RecognitionCallback interface {
	OnOpen()
	OnComplete()
	OnError(err error)
	OnClose()
	OnEvent(result *RecognitionResult)
}

// RecognitionResult represents the result of a speech recognition event.
type RecognitionResult struct {
	Response  map[string]interface{} // Full response payload
	Sentences []interface{}          // Recognition sentences
	Usage     interface{}            // Usage statistics
}

// Recognition handles real-time speech recognition.
type Recognition struct {
	Model      string
	APIKey     string
	Format     string
	SampleRate int
	Workspace  string
	conn       *websocket.Conn
	callback   RecognitionCallback
	mu         sync.Mutex
	running    bool
	taskID     string
	wg         sync.WaitGroup
}

// NewRecognition creates a new recognition client.
func NewRecognition(model, apiKey, format string, sampleRate int) *Recognition {
	return &Recognition{
		Model:      model,
		APIKey:     apiKey,
		Format:     format,
		SampleRate: sampleRate,
	}
}

// Start starts the recognition session.
func (r *Recognition) Start(ctx context.Context, callback RecognitionCallback, params map[string]interface{}) error {
	if r.running {
		return errors.New("recognition already started")
	}
	r.callback = callback

	// Generate Task ID
	r.taskID = strings.ReplaceAll(uuid.New().String(), "-", "")

	// WebSocket URL
	url := BaseWebsocketURL

	header := http.Header{}
	header.Set("Authorization", "Bearer "+r.APIKey)
	if r.Workspace != "" {
		header.Set("X-DashScope-Workspace", r.Workspace)
	}

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return err
	}
	r.conn = conn
	r.running = true

	// Construct Initial Request
	reqParams := make(map[string]interface{})
	reqParams["sample_rate"] = r.SampleRate
	reqParams["format"] = r.Format
	for k, v := range params {
		reqParams[k] = v
	}

	reqPayload := map[string]interface{}{
		"header": map[string]interface{}{
			"action":    "run-task",
			"task_id":   r.taskID,
			"streaming": "duplex",
		},
		"payload": map[string]interface{}{
			"task_group": "audio",
			"task":       "asr",
			"function":   "recognition",
			"model":      r.Model,
			"parameters": reqParams,
			"input":      map[string]interface{}{}, // Required input field
		},
	}

	if err := conn.WriteJSON(reqPayload); err != nil {
		conn.Close()
		r.running = false
		return err
	}

	r.callback.OnOpen()

	// Start reading loop
	r.wg.Add(1)
	go r.readLoop()

	return nil
}

// Stop stops the recognition session.
func (r *Recognition) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}

	// Send finish-task message
	finishPayload := map[string]interface{}{
		"header": map[string]interface{}{
			"action":    "finish-task",
			"task_id":   r.taskID,
			"streaming": "duplex",
		},
		"payload": map[string]interface{}{
			"input": map[string]interface{}{},
		},
	}
	r.conn.WriteJSON(finishPayload)
	r.mu.Unlock()

	// Wait for readLoop to finish (which happens on task-finished or error)
	// Add timeout to prevent hanging
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Normal finish
	case <-time.After(10 * time.Second):
		// Timeout, force close
		r.mu.Lock()
		if r.conn != nil {
			r.conn.Close()
		}
		r.mu.Unlock()
	}

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()
}

// SendAudioFrame sends a chunk of audio data.
func (r *Recognition) SendAudioFrame(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return errors.New("recognition not running")
	}
	return r.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (r *Recognition) readLoop() {
	defer r.wg.Done()
	defer func() {
		if r.callback != nil {
			r.callback.OnClose()
		}
	}()

	for {
		messageType, message, err := r.conn.ReadMessage()
		if err != nil {
			if r.running && r.callback != nil {
				// Only report error if we didn't intentionally close
				if !strings.Contains(err.Error(), "use of closed network connection") {
					r.callback.OnError(err)
				}
			}
			return
		}

		if messageType == websocket.TextMessage {
			var resp map[string]interface{}
			if err := json.Unmarshal(message, &resp); err != nil {
				if r.callback != nil {
					r.callback.OnError(err)
				}
				continue
			}

			header, ok := resp["header"].(map[string]interface{})
			if !ok {
				continue
			}

			event := header["event"].(string)
			if event == "task-failed" {
				if r.callback != nil {
					r.callback.OnError(fmt.Errorf("task failed: %v", header["error_message"]))
				}
				return // Stop on failure
			} else if event == "task-finished" {
				if r.callback != nil {
					r.callback.OnComplete()
				}
				// Don't return yet, might be more messages?
				// Usually task-finished is the end.
				return
			} else if event == "result-generated" {
				payload, ok := resp["payload"].(map[string]interface{})
				if ok {
					result := &RecognitionResult{
						Response: resp,
					}

					if output, ok := payload["output"].(map[string]interface{}); ok {
						if sentence, ok := output["sentence"].(map[string]interface{}); ok {
							result.Sentences = append(result.Sentences, sentence)
						}
					}

					if usage, ok := payload["usage"]; ok {
						result.Usage = usage
					}

					if r.callback != nil {
						r.callback.OnEvent(result)
					}
				}
			}
		}
	}
}
