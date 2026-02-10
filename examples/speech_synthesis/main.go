/*
 * @Author: Ceoifung
 * @Date: 2026-02-10 19:38:51
 * @LastEditors: Ceoifung
 * @LastEditTime: 2026-02-10 19:38:54
 * @Description: XiaoRGEEK All Rights Reserved. Powered By Ceoifung
 */
package main

import (
	"context"
	"fmt"
	"go-dashcope/dashscope"
	"os"
)

type MyCallback struct {
	file *os.File
}

func (c *MyCallback) OnOpen() {
	fmt.Println("WebSocket Open")
	var err error
	c.file, err = os.Create("output.wav")
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
	}
}

func (c *MyCallback) OnComplete() {
	fmt.Println("Task Complete")
}

func (c *MyCallback) OnError(err error) {
	fmt.Printf("Error: %v\n", err)
}

func (c *MyCallback) OnClose() {
	fmt.Println("WebSocket Closed")
	if c.file != nil {
		c.file.Close()
	}
}

func (c *MyCallback) OnEvent(result *dashscope.SpeechSynthesisResult) {
	if len(result.AudioFrame) > 0 {
		fmt.Printf("Received audio data: %d bytes\n", len(result.AudioFrame))
		if c.file != nil {
			c.file.Write(result.AudioFrame)
		}
	}
	if result.Response != nil {
		fmt.Printf("Received meta data: %v\n", result.Response)
	}
}

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set DASHSCOPE_API_KEY environment variable")
		return
	}

	synthesizer := dashscope.NewSpeechSynthesizer("sambert-zhichu-v1", apiKey)
	callback := &MyCallback{}

	params := map[string]interface{}{
		"format":      dashscope.AudioFormatWAV,
		"sample_rate": 48000,
	}

	fmt.Println("Synthesizing...")
	_, err := synthesizer.Call(context.Background(), "你好，阿里云！", callback, params)
	if err != nil {
		fmt.Printf("Call failed: %v\n", err)
	}

	// Example: Synchronous call (saving to file directly)
	fmt.Println("Synthesizing synchronously...")
	syncParams := map[string]interface{}{
		"format":      dashscope.AudioFormatWAV,
		"sample_rate": 48000,
	}
	// Using the same synthesizer
	result, err := synthesizer.Call(context.Background(), "欢迎使用DashScope Go SDK", nil, syncParams)
	if err != nil {
		fmt.Printf("Sync Call failed: %v\n", err)
	} else {
		fmt.Println("Sync Call success")
		if err := result.Save("output_sync.wav"); err != nil {
			fmt.Printf("Save failed: %v\n", err)
		} else {
			fmt.Println("Saved to output_sync.wav")
		}
		if result.Usage != nil {
			fmt.Printf("Usage: %d characters\n", result.Usage.Characters)
		}
	}
}
