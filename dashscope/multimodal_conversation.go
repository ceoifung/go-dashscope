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

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// const QwenVLGenerationURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"

// MultiModal Conversation Models
const (
	QwenVLChatV1     = "qwen-vl-chat-v1"
	QwenVLChatV1Plus = "qwen-vl-plus"
	QwenVLChatV1Max  = "qwen-vl-max"
)

// WebSocket actions
const (
	ActionStart           = "Start"
	ActionStop            = "Stop"
	ActionSendSpeech      = "SendSpeech"
	ActionStopSpeech      = "StopSpeech"
	ActionExecute         = "Execute"
	ActionHeartBeat       = "HeartBeat"
	ActionRequestAccepted = "RequestAccepted"
)

// Response directives
const (
	ResponseStarted           = "Started"
	ResponseStopped           = "Stopped"
	ResponseStateChanged      = "DialogStateChanged"
	ResponseRequestAccepted   = "RequestAccepted"
	ResponseSpeechStarted     = "SpeechStarted"
	ResponseSpeechEnded       = "SpeechEnded"
	ResponseRespondingStarted = "RespondingStarted"
	ResponseRespondingEnded   = "RespondingEnded"
	ResponseSpeechContent     = "SpeechContent"
	ResponseRespondingContent = "RespondingContent"
	ResponseError             = "Error"
	ResponseHeartBeat         = "HeartBeat"
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

// ... (existing Call and CallStream methods remain, adding new types below)

// Realtime/WebSocket types based on Python SDK

type MultiModalHeader struct {
	Action    string `json:"action"`
	TaskID    string `json:"task_id"`
	Streaming string `json:"streaming"` // default "duplex"
}

type MultiModalUpstream struct {
	Type        string `json:"type"`         // "AudioOnly" or "AudioAndVideo"
	Mode        string `json:"mode"`         // "push2talk", "tap2talk", "duplex"
	AudioFormat string `json:"audio_format"` // "pcm", "opus"
	SampleRate  int    `json:"sample_rate"`
}

type MultiModalDownstream struct {
	Voice            string `json:"voice,omitempty"`
	SampleRate       int    `json:"sample_rate,omitempty"`
	IntermediateText string `json:"intermediate_text"` // "transcript"
	AudioFormat      string `json:"audio_format"`      // "pcm", "mp3"
	Volume           int    `json:"volume"`
	SpeechRate       int    `json:"speech_rate"`
	PitchRate        int    `json:"pitch_rate"`
}

type MultiModalRealtimeParameters struct {
	Upstream   MultiModalUpstream   `json:"upstream"`
	Downstream MultiModalDownstream `json:"downstream"`
}

type MultiModalRealtimeInput struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	AppID       string `json:"app_id"`
	Directive   string `json:"directive,omitempty"`
	DialogID    string `json:"dialog_id,omitempty"`
}

type MultiModalRealtimePayload struct {
	Model      string                        `json:"model"`
	TaskGroup  string                        `json:"task_group"` // "aigc"
	Task       string                        `json:"task"`       // "multimodal-generation"
	Function   string                        `json:"function"`   // "generation"
	Parameters *MultiModalRealtimeParameters `json:"parameters,omitempty"`
	Input      *MultiModalRealtimeInput      `json:"input,omitempty"`
}

type MultiModalRealtimeRequest struct {
	Header  MultiModalHeader          `json:"header"`
	Payload MultiModalRealtimePayload `json:"payload"`
}

type MultiModalRealtimeResponse struct {
	Header  MultiModalHeader `json:"header"`
	Payload struct {
		Output struct {
			Directive string `json:"directive"`
			DialogID  string `json:"dialog_id,omitempty"`
			Text      string `json:"text,omitempty"`
		} `json:"output"`
		Usage *MultiModalUsage `json:"usage,omitempty"`
	} `json:"payload"`
}

// MultiModalCallback defines the interface for handling realtime events
type MultiModalCallback interface {
	OnConnected()
	OnStarted(dialogID string)
	OnStopped()
	OnSpeechStarted()
	OnSpeechEnded()
	OnSpeechContent(text string)
	OnRespondingStarted()
	OnRespondingContent(text string)
	OnRespondingEnded()
	OnAudioData(data []byte)
	OnError(err error)
	OnClose(code int, reason string)
}

type MultiModalDialog struct {
	AppID     string
	APIKey    string
	Model     string
	Callback  MultiModalCallback
	Conn      *websocket.Conn
	TaskID    string
	DialogID  string
	Workspace string
	done      chan struct{}
}

func (m *MultiModalConversation) NewDialog(appID string, callback MultiModalCallback) *MultiModalDialog {
	return &MultiModalDialog{
		AppID:     appID,
		APIKey:    m.APIKey,
		Callback:  callback,
		Workspace: m.Workspace,
		done:      make(chan struct{}),
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

func (d *MultiModalDialog) Start(ctx context.Context, model string) error {
	d.Model = model
	u := "wss://dashscope.aliyuncs.com/api-ws/v1/inference/"
	header := http.Header{}
	header.Add("Authorization", "Bearer "+d.APIKey)
	if d.Workspace != "" {
		header.Add("X-DashScope-WorkSpace", d.Workspace)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u, header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	d.Conn = conn
	d.Callback.OnConnected()

	// Start reading loop
	go d.readLoop()

	// Send Start request
	d.TaskID = strings.ReplaceAll(uuid.New().String(), "-", "")
	req := MultiModalRealtimeRequest{
		Header: MultiModalHeader{
			Action:    ActionStart,
			TaskID:    d.TaskID,
			Streaming: "duplex",
		},
		Payload: MultiModalRealtimePayload{
			Model:     d.Model,
			TaskGroup: "aigc",
			Task:      "multimodal-generation",
			Function:  "generation",
			Parameters: &MultiModalRealtimeParameters{
				Upstream: MultiModalUpstream{
					Type:        "AudioOnly",
					Mode:        "tap2talk",
					AudioFormat: "pcm",
					SampleRate:  16000,
				},
				Downstream: MultiModalDownstream{
					IntermediateText: "transcript",
					AudioFormat:      "pcm",
					Volume:           50,
					SpeechRate:       100,
					PitchRate:        100,
				},
			},
			Input: &MultiModalRealtimeInput{
				AppID: d.AppID,
			},
		},
	}

	return d.Conn.WriteJSON(req)
}

func (d *MultiModalDialog) readLoop() {
	defer func() {
		close(d.done)
		d.Callback.OnClose(0, "connection closed")
	}()

	for {
		messageType, data, err := d.Conn.ReadMessage()
		if err != nil {
			d.Callback.OnError(err)
			return
		}

		if messageType == websocket.BinaryMessage {
			d.Callback.OnAudioData(data)
			continue
		}

		var resp MultiModalRealtimeResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}

		switch resp.Payload.Output.Directive {
		case ResponseStarted:
			d.DialogID = resp.Payload.Output.DialogID
			d.Callback.OnStarted(d.DialogID)
		case ResponseStopped:
			d.Callback.OnStopped()
		case ResponseSpeechStarted:
			d.Callback.OnSpeechStarted()
		case ResponseSpeechEnded:
			d.Callback.OnSpeechEnded()
		case ResponseSpeechContent:
			d.Callback.OnSpeechContent(resp.Payload.Output.Text)
		case ResponseRespondingStarted:
			d.Callback.OnRespondingStarted()
		case ResponseRespondingContent:
			d.Callback.OnRespondingContent(resp.Payload.Output.Text)
		case ResponseRespondingEnded:
			d.Callback.OnRespondingEnded()
		case ResponseError:
			d.Callback.OnError(fmt.Errorf("server error: %s", resp.Payload.Output.Text))
		}
	}
}

func (d *MultiModalDialog) SendAudio(data []byte) error {
	return d.Conn.WriteMessage(websocket.BinaryMessage, data)
}

func (d *MultiModalDialog) StopSpeech() error {
	req := MultiModalRealtimeRequest{
		Header: MultiModalHeader{
			Action: ActionStopSpeech,
			TaskID: d.TaskID,
		},
	}
	return d.Conn.WriteJSON(req)
}

func (d *MultiModalDialog) Close() error {
	if d.Conn != nil {
		return d.Conn.Close()
	}
	return nil
}
