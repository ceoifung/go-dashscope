package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ceoifung/go-dashcope/dashscope"
	"github.com/ceoifung/go-dashcope/examples/audio"
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

	fmt.Println("\n--- Auto Recognition Test (3 seconds) ---")

	rec := dashscope.NewRecognition("paraformer-realtime-v1", apiKey, "pcm", 16000)
	cb := &MyRecognitionCallback{}
	err := rec.Start(context.Background(), cb, nil)
	if err != nil {
		fmt.Println("Start Error:", err)
		return
	}
	fmt.Println("Recognition Started")

	recorder, err := audio.NewRecorder()
	if err != nil {
		fmt.Println("Recorder Init Error:", err)
		rec.Stop()
		return
	}

	if err := recorder.Start(); err != nil {
		fmt.Println("Recorder Start Error:", err)
		recorder.Close()
		rec.Stop()
		return
	}
	fmt.Println("Recorder Started")

	// Read loop
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 1280) // 40ms
		totalBytes := 0
		for {
			select {
			case <-done:
				return
			default:
				n, err := recorder.Read(buf)
				if n > 0 {
					if sendErr := rec.SendAudioFrame(buf[:n]); sendErr != nil {
						fmt.Printf("Send Error: %v\n", sendErr)
						break
					}
					totalBytes += n
				}
				if err != nil {
					if err != io.EOF {
						fmt.Printf("Recorder Read Error: %v\n", err)
					}
					break
				}
			}
		}
		// fmt.Printf("Total audio sent: %d bytes\n", totalBytes)
	}()

	time.Sleep(3 * time.Second)
	fmt.Println("Stopping...")

	close(done)
	recorder.Close()
	rec.Stop()
	fmt.Println("Stopped. Waiting for final result...")
	time.Sleep(1 * time.Second) // Wait for callback
}
