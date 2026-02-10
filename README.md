# Alibaba DashScope Go SDK

[English](README.md) | [中文](README_CN.md)

This is a Go SDK for Alibaba Cloud DashScope (ModelScope) API. It allows you to easily interact with DashScope services including Text Generation (Qwen), Image Synthesis (Wanx), Audio Transcription (Paraformer), Text-to-Speech (Sambert), Embeddings, and more.

## Features

- **Text Generation**: Support for Qwen (Tongyi Qianwen) models with streaming and synchronous modes.
- **Image Synthesis**: Support for Wanx (Tongyi Wanxiang) models.
- **Audio Recognition (ASR)**: Real-time speech recognition using WebSocket (Paraformer).
- **Audio Transcription**: File-based audio transcription.
- **Text-to-Speech (TTS)**: Speech synthesis using Sambert models.
- **Multimodal Conversation**: Support for Qwen-VL (Visual Language) models.
- **NLU & Embeddings**: Text embedding, reranking, and natural language understanding.
- **Context Support**: Full `context.Context` support for timeout and cancellation.
- **Pure Go**: No CGO dependencies for audio recording (uses `go-wca` on Windows).

## Installation

```bash
go get github.com/yourusername/go-dashscope
```
*(Note: Replace `github.com/yourusername/go-dashscope` with the actual repository path if hosted remotely)*

## Configuration

Set your DashScope API key as an environment variable:

```powershell
# Windows PowerShell
$env:DASHSCOPE_API_KEY = "your-api-key"

# Linux/macOS
export DASHSCOPE_API_KEY="your-api-key"
```

Or pass it directly when creating a client.

## Usage Examples

### Text Generation (Qwen)

```go
package main

import (
	"context"
	"fmt"
	"go-dashcope/dashscope"
)

func main() {
	gen := dashscope.NewGeneration("") // Uses env DASHSCOPE_API_KEY

	req := dashscope.GenerationRequest{
		Model: dashscope.QwenTurbo,
		Input: dashscope.GenerationInput{
			Prompt: "Hello, tell me a joke.",
		},
	}

	resp, err := gen.Call(context.Background(), req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Output.Text)
}
```

### Multimodal Conversation (Qwen-VL)

```go
conv := dashscope.NewMultiModalConversation("")

req := dashscope.MultiModalConversationRequest{
    Model: dashscope.QwenVLChatV1,
    Input: dashscope.MultiModalConversationInput{
        Messages: []dashscope.MultiModalMessage{
            {
                Role: "user",
                Content: []dashscope.MultiModalContentItem{
                    {Image: "https://example.com/image.png"},
                    {Text: "What is in this image?"},
                },
            },
        },
    },
}

resp, err := conv.Call(context.Background(), req)
if err != nil {
    panic(err)
}

fmt.Println(resp.Output.Choices[0].Message.Content[0].Text)
```

### Image Synthesis (Wanx)

```go
imgSynth := dashscope.NewImageSynthesis("")

req := dashscope.ImageSynthesisRequest{
    Model: dashscope.WanxV1,
    Input: dashscope.ImageSynthesisInput{
        Prompt: "A futuristic city in cyberpunk style",
    },
    Parameters: &dashscope.ImageSynthesisParameters{
        Size: "1024*1024",
        N:    1,
    },
}

resp, err := imgSynth.Call(context.Background(), req)
if err != nil {
    panic(err)
}

for _, res := range resp.Output.Results {
    fmt.Println("Image URL:", res.URL)
}
```

### Real-time Speech Recognition (ASR)

```go
rec := dashscope.NewRecognition("paraformer-realtime-v1", "", "pcm", 16000)
callback := &MyRecognitionCallback{} // Implement RecognitionCallback interface

err := rec.Start(context.Background(), callback, nil)
if err != nil {
    panic(err)
}

// Send audio data...
rec.SendAudioFrame(audioData)

rec.Stop()
```

### Audio Transcription (File)

```go
trans := dashscope.NewTranscription("")

req := dashscope.TranscriptionRequest{
    Model: dashscope.ParaformerV1,
    Input: dashscope.TranscriptionInput{
        FileURLs: []string{"https://example.com/audio.wav"},
    },
}

resp, err := trans.Call(context.Background(), req)
if err != nil {
    panic(err)
}

// Results will be in resp.Output.Results (JSON raw message)
fmt.Println(string(resp.Output.Results))
```

### Text-to-Speech (TTS)

```go
tts := dashscope.NewSpeechSynthesizer("sambert-zhichu-v1", "")

params := map[string]interface{}{
    "format":      "wav",
    "sample_rate": 48000,
}

// Simple synchronous call
result, err := tts.Call(context.Background(), "Hello, DashScope!", nil, params)
if err != nil {
    panic(err)
}

if err := result.Save("output.wav"); err != nil {
    fmt.Printf("Save failed: %v\n", err)
}
```

### Text Embeddings

```go
emb := dashscope.NewTextEmbedding("")

req := dashscope.TextEmbeddingRequest{
    Model: dashscope.TextEmbeddingV1,
    Input: dashscope.TextEmbeddingInput{
        Texts: []string{"Hello world", "Machine learning"},
    },
}

resp, err := emb.Call(context.Background(), req)
if err != nil {
    panic(err)
}

for _, res := range resp.Output.Embeddings {
    fmt.Printf("Index: %d, Embedding: %v\n", res.TextIndex, res.Embedding[:5])
}
```

### Text ReRank

```go
rerank := dashscope.NewTextReRank("")

req := dashscope.TextReRankRequest{
    Model: dashscope.GteRerank,
    Input: dashscope.TextReRankInput{
        Query:     "What is DashScope?",
        Documents: []string{"DashScope is a model service.", "Go is a programming language."},
    },
}

resp, err := rerank.Call(context.Background(), req)
if err != nil {
    panic(err)
}

for _, res := range resp.Output.Results {
    fmt.Printf("Doc Index: %d, Score: %f\n", res.Index, res.RelevanceScore)
}
```

### Natural Language Understanding (NLU)

```go
nlu := dashscope.NewUnderstanding("")

req := dashscope.UnderstandingRequest{
    Model: dashscope.OpenNLUV1,
    Input: dashscope.UnderstandingInput{
        Sentence: "What's the weather in Beijing today?",
        Labels:   "Weather,Traffic,Sports",
        Task:     "classification",
    },
}

resp, err := nlu.Call(context.Background(), req)
if err != nil {
    panic(err)
}

fmt.Println(string(resp.Output))
```

### Complete Examples

You can find complete, self-contained examples for each feature in the `examples/` directory:

- [Text Generation](examples/text_generation/main.go)
- [Image Synthesis](examples/image_synthesis/main.go)
- [Speech Recognition](examples/speech_recognition/main.go)
- [Speech Synthesis](examples/speech_synthesis/main.go)
- [Audio Transcription](examples/transcription/main.go)
- [Multimodal Conversation](examples/multimodal_conversation/main.go)
- [Text Embedding](examples/text_embedding/main.go)
- [Text ReRank](examples/rerank/main.go)
- [NLU](examples/nlu/main.go)
- [Interactive Recognition](examples/interactive_recognition/main.go)
- [Auto Recognition](examples/auto_recognition/main.go)
- [Check Audio](examples/check_audio/main.go)

## License

MIT License
