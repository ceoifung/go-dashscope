package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ceoifung/go-dashscope/dashscope"
	"github.com/ceoifung/go-dashscope/examples/audio"
	"github.com/eiannone/keyboard"
)

type RealtimeCallback struct {
	Player   *audio.Player
	Recorder *audio.Recorder
}

func (c *RealtimeCallback) OnConnected() {
	fmt.Println("\n[System] 🟢 Connected to Realtime Server")
}

func (c *RealtimeCallback) OnStarted(dialogID string) {
	fmt.Printf("[System] 🚀 Dialog Started (ID: %s)\n", dialogID)
}

func (c *RealtimeCallback) OnStopped() {
	fmt.Println("[System] 🛑 Dialog Stopped")
}

func (c *RealtimeCallback) OnSpeechStarted() {
	fmt.Print("\n[User] 🎙️  Speaking...")
}

func (c *RealtimeCallback) OnSpeechEnded() {
	fmt.Print(" (Ended)")
}

func (c *RealtimeCallback) OnSpeechContent(text string) {
	fmt.Printf("\r[User] 🎙️  %s", text)
}

func (c *RealtimeCallback) OnRespondingStarted() {
	fmt.Print("\n[AI] 🤖 Thinking...")
}

func (c *RealtimeCallback) OnRespondingContent(text string) {
	fmt.Printf("\r[AI] 🤖 %s", text)
}

func (c *RealtimeCallback) OnRespondingEnded() {
	fmt.Println()
}

func (c *RealtimeCallback) OnAudioData(data []byte) {
	if c.Player != nil {
		c.Player.Play(data)
	}
}

func (c *RealtimeCallback) OnError(err error) {
	fmt.Printf("\n[Error] ❌ %v\n", err)
}

func (c *RealtimeCallback) OnClose(code int, reason string) {
	fmt.Printf("\n[System] 🔌 Closed: %s (%d)\n", reason, code)
}

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		log.Fatal("DASHSCOPE_API_KEY environment variable is not set")
	}

	appID := "YOUR_APP_ID" // 需要用户替换为实际的应用ID
	if id := os.Getenv("DASHSCOPE_APP_ID"); id != "" {
		appID = id
	}

	player, err := audio.NewPlayer()
	if err != nil {
		log.Fatalf("Failed to initialize audio player: %v", err)
	}
	defer player.Close()

	recorder, err := audio.NewRecorder()
	if err != nil {
		log.Fatalf("Failed to initialize audio recorder: %v", err)
	}
	defer recorder.Close()

	mm := dashscope.NewMultiModalConversation(apiKey)
	callback := &RealtimeCallback{
		Player:   player,
		Recorder: recorder,
	}
	dialog := mm.NewDialog(appID, callback)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := dialog.Start(ctx, dashscope.QwenVLChatV1Plus); err != nil {
		log.Fatalf("Failed to start dialog: %v", err)
	}
	defer dialog.Close()

	if err := keyboard.Open(); err != nil {
		log.Fatal(err)
	}
	defer keyboard.Close()

	fmt.Println("\n=================================================")
	fmt.Println("   Multimodal Realtime Voice Call (Qwen-VL)")
	fmt.Println("=================================================")
	fmt.Println("Controls:")
	fmt.Println("  [SPACE] : Press to Start/Stop Speaking")
	fmt.Println("  [ESC]   : Quit")
	fmt.Println("-------------------------------------------------")

	isRecording := false
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			char, key, err := keyboard.GetKey()
			if err != nil {
				break
			}
			if key == keyboard.KeyEsc {
				cancel()
				return
			}
			if key == keyboard.KeySpace || char == ' ' {
				if !isRecording {
					isRecording = true
					fmt.Print("\n[State] 🔴 Recording... (Press SPACE to stop)")

					// 重新创建 Recorder 因为之前的可能已经 Close 了
					var err error
					recorder, err = audio.NewRecorder()
					if err != nil {
						fmt.Printf("\n[Error] Failed to re-init recorder: %v\n", err)
						isRecording = false
						continue
					}

					recorder.Start()
					go func() {
						buf := make([]byte, 3200) // 100ms
						for isRecording {
							n, err := recorder.Read(buf)
							if err != nil {
								break
							}
							if n > 0 {
								dialog.SendAudio(buf[:n])
							}
						}
					}()
				} else {
					isRecording = false
					dialog.StopSpeech()
					recorder.Close()
					fmt.Print("\n[State] ⚪ Stopped.")
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-sigChan:
	}
	fmt.Println("\nExiting...")
}
