package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ceoifung/go-dashscope/dashscope"
)

type MyRecognitionCallback struct{}

func (c *MyRecognitionCallback) OnOpen() {
	fmt.Println("Recognition Open")
}

func (c *MyRecognitionCallback) OnComplete() {
	fmt.Println("Recognition Complete")
}

func (c *MyRecognitionCallback) OnError(err error) {
	fmt.Printf("Recognition Error: %v\n", err)
}

func (c *MyRecognitionCallback) OnClose() {
	fmt.Println("Recognition Closed")
}

func (c *MyRecognitionCallback) OnEvent(result *dashscope.RecognitionResult) {
	if len(result.Sentences) > 0 {
		for _, s := range result.Sentences {
			if sentMap, ok := s.(map[string]interface{}); ok {
				fmt.Printf("Text: %s (End: %v)\n", sentMap["text"], sentMap["end_time"])
			}
		}
	}
}

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set DASHSCOPE_API_KEY environment variable")
		return
	}

	// Download sample audio
	audioURL := "https://dashscope.oss-cn-beijing.aliyuncs.com/samples/audio/paraformer/hello_world.wav"
	resp, err := http.Get(audioURL)
	if err != nil {
		fmt.Printf("Failed to download audio: %v\n", err)
		return
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read audio: %v\n", err)
		return
	}

	// Create Recognition client
	// paraformer-realtime-v1 sample rate is usually 16000
	rec := dashscope.NewRecognition("paraformer-realtime-v1", apiKey, "pcm", 16000)
	callback := &MyRecognitionCallback{}

	fmt.Println("\n--- Testing Real-time Recognition ---")
	err = rec.Start(context.Background(), callback, nil)
	if err != nil {
		fmt.Printf("Start failed: %v\n", err)
		return
	}

	// Stream audio
	chunkSize := 1280 // 40ms for 16kHz
	for i := 0; i < len(audioData); i += chunkSize {
		end := i + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}

		err := rec.SendAudioFrame(audioData[i:end])
		if err != nil {
			fmt.Printf("Send failed: %v\n", err)
			break
		}
		time.Sleep(40 * time.Millisecond) // Simulate real-time
	}

	rec.Stop()
}
