package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ceoifung/go-dashscope/dashscope"
	"github.com/ceoifung/go-dashscope/examples/audio"

	"github.com/eiannone/keyboard"
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

	if err := keyboard.Open(); err != nil {
		fmt.Println("Keyboard Error:", err)
		return
	}
	defer keyboard.Close()

	fmt.Println("\n--- Interactive Real-time Recognition ---")
	fmt.Println("Press SPACE to start/stop recording. Press ESC to quit.")

	var rec *dashscope.Recognition
	var recording bool
	var mu sync.Mutex
	var recorder *audio.Recorder
	var lastPressTime time.Time

	for {
		_, key, err := keyboard.GetKey()
		if err != nil {
			break
		}
		if key == keyboard.KeyEsc {
			break
		}
		if key == keyboard.KeySpace {
			if time.Since(lastPressTime) < 500*time.Millisecond {
				continue
			}
			lastPressTime = time.Now()

			mu.Lock()
			if !recording {
				fmt.Println("Starting recording...")

				// Start Recognition
				rec = dashscope.NewRecognition("paraformer-realtime-v1", apiKey, "pcm", 16000)
				cb := &MyRecognitionCallback{}
				err := rec.Start(context.Background(), cb, nil)
				if err != nil {
					fmt.Println("Start Recognition Error:", err)
					mu.Unlock()
					continue
				}

				// Start Recorder
				recorder, err = audio.NewRecorder()
				if err != nil {
					fmt.Println("Start Recorder Error:", err)
					rec.Stop()
					mu.Unlock()
					continue
				}

				if err := recorder.Start(); err != nil {
					fmt.Println("Recorder Start Error:", err)
					recorder.Close()
					rec.Stop()
					mu.Unlock()
					continue
				}

				recording = true
				fmt.Println("Recording started. Speak now...")

				// Start reading audio and sending
				go func() {
					buf := make([]byte, 3200) // 100ms
					totalBytes := 0
					for {
						n, err := recorder.Read(buf)
						if n > 0 {
							// fmt.Printf(".") // Visual feedback
							if sendErr := rec.SendAudioFrame(buf[:n]); sendErr != nil {
								fmt.Printf("\nSend Error: %v\n", sendErr)
								break
							}
							totalBytes += n
						}
						if err != nil {
							if err != io.EOF {
								fmt.Printf("\nRecorder Read Error: %v\n", err)
							}
							break
						}
					}
					fmt.Printf("\nTotal audio sent: %d bytes\n", totalBytes)
				}()

			} else {
				fmt.Println("Stopping recording...")
				recording = false

				// Stop Recorder
				if recorder != nil {
					recorder.Close()
				}

				// Stop Recognition
				if rec != nil {
					rec.Stop()
				}
				fmt.Println("Stopped. Waiting for final result...")
			}
			mu.Unlock()
		}
	}
}
