# 阿里云 DashScope Go SDK

[English](README.md) | [中文](README_CN.md)

这是阿里云 DashScope (模型服务灵积) API 的 Go 语言 SDK。它允许您轻松集成 DashScope 的各项服务，包括文本生成 (通义千问)、图像合成 (通义万相)、语音识别 (Paraformer)、语音合成 (Sambert)、文本向量 (Embeddings) 等。

## 功能特性

- **文本生成**: 支持通义千问 (Qwen) 模型，支持流式 (Streaming) 和同步模式。
- **图像合成**: 支持通义万相 (Wanx) 模型。
- **语音识别 (ASR)**: 基于 WebSocket 的实时语音识别 (Paraformer)。
- **语音转写**: 基于文件的音频转写服务。
- **语音合成 (TTS)**: 使用 Sambert 模型进行语音合成。
- **多模态对话**: 支持 Qwen-VL (视觉语言) 模型。
- **NLU & 向量**: 支持文本 Embedding、重排序 (Rerank) 和自然语言理解。
- **Context 支持**: 全面支持 `context.Context`，可控制超时和取消。
- **纯 Go 实现**: 音频录制无 CGO 依赖 (Windows 下使用 `go-wca`)。

## 安装

```bash
go get github.com/ceoifung/go-dashscope
```

## 配置

请将您的 DashScope API Key 设置为环境变量：

```powershell
# Windows PowerShell
$env:DASHSCOPE_API_KEY = "your-api-key"

# Linux/macOS
export DASHSCOPE_API_KEY="your-api-key"
```

或者在创建客户端时直接传入。

## 使用示例

### 文本生成 (通义千问)

```go
package main

import (
	"context"
	"fmt"
	"github.com/ceoifung/go-dashscope/dashscope"
)

func main() {
	gen := dashscope.NewGeneration("") // 使用环境变量 DASHSCOPE_API_KEY

	req := dashscope.GenerationRequest{
		Model: dashscope.QwenTurbo,
		Input: dashscope.GenerationInput{
			Prompt: "你好，讲个笑话。",
		},
	}

	resp, err := gen.Call(context.Background(), req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Output.Text)
}
```

### 多模态对话 (Qwen-VL)

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
                    {Text: "图片里有什么？"},
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

### 图像合成 (通义万相)

```go
imgSynth := dashscope.NewImageSynthesis("")

req := dashscope.ImageSynthesisRequest{
    Model: dashscope.WanxV1,
    Input: dashscope.ImageSynthesisInput{
        Prompt: "一只赛博朋克风格的未来城市猫咪",
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

### 实时语音识别 (ASR)

```go
rec := dashscope.NewRecognition("paraformer-realtime-v1", "", "pcm", 16000)
callback := &MyRecognitionCallback{} // 实现 RecognitionCallback 接口

err := rec.Start(context.Background(), callback, nil)
if err != nil {
    panic(err)
}

// 发送音频数据...
rec.SendAudioFrame(audioData)

rec.Stop()
```

### 语音文件转写

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

// 结果在 resp.Output.Results (JSON raw message)
fmt.Println(string(resp.Output.Results))
```

### 语音合成 (TTS)

```go
tts := dashscope.NewSpeechSynthesizer("sambert-zhichu-v1", "")

params := map[string]interface{}{
    "format":      "wav",
    "sample_rate": 48000,
}

// 简单的同步调用
result, err := tts.Call(context.Background(), "你好，阿里云！", nil, params)
if err != nil {
    panic(err)
}

if err := result.Save("output.wav"); err != nil {
    fmt.Printf("保存失败: %v\n", err)
}
```

### 文本向量 (Embeddings)

```go
emb := dashscope.NewTextEmbedding("")

req := dashscope.TextEmbeddingRequest{
    Model: dashscope.TextEmbeddingV1,
    Input: dashscope.TextEmbeddingInput{
        Texts: []string{"你好，世界", "机器学习"},
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

### 文本重排序 (ReRank)

```go
rerank := dashscope.NewTextReRank("")

req := dashscope.TextReRankRequest{
    Model: dashscope.GteRerank,
    Input: dashscope.TextReRankInput{
        Query:     "什么是DashScope？",
        Documents: []string{"DashScope是一个模型服务。", "Go是一门编程语言。"},
    },
}

resp, err := rerank.Call(context.Background(), req)
if err != nil {
    panic(err)
}

for _, res := range resp.Output.Results {
    fmt.Printf("文档索引: %d, 相关性得分: %f\n", res.Index, res.RelevanceScore)
}
```

### 自然语言理解 (NLU)

```go
nlu := dashscope.NewUnderstanding("")

req := dashscope.UnderstandingRequest{
    Model: dashscope.OpenNLUV1,
    Input: dashscope.UnderstandingInput{
        Sentence: "北京今天天气怎么样？",
        Labels:   "天气,交通,体育",
        Task:     "classification",
    },
}

resp, err := nlu.Call(context.Background(), req)
if err != nil {
    panic(err)
}

fmt.Println(string(resp.Output))
```

### 完整示例代码

您可以在 `examples/` 目录下找到每个功能的完整独立示例代码：

- [文本生成](examples/text_generation/main.go)
- [图像合成](examples/image_synthesis/main.go)
- [实时语音识别](examples/speech_recognition/main.go)
- [语音合成](examples/speech_synthesis/main.go)
- [音频转写](examples/transcription/main.go)
- [多模态对话](examples/multimodal_conversation/main.go)
- [文本向量](examples/text_embedding/main.go)
- [文本重排序](examples/rerank/main.go)
- [自然语言理解 (NLU)](examples/nlu/main.go)
- [交互式语音识别](examples/interactive_recognition/main.go)
- [自动语音识别](examples/auto_recognition/main.go)
- [音频设备检测](examples/check_audio/main.go)

## 许可证

MIT License
