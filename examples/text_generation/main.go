package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ceoifung/go-dashcope/dashscope"
)

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set DASHSCOPE_API_KEY environment variable")
		return
	}

	gen := dashscope.NewGeneration(apiKey)

	// Test 1: Simple text generation
	req := dashscope.GenerationRequest{
		Model: dashscope.QwenTurbo,
		Input: dashscope.GenerationInput{
			Prompt: "你好，请介绍一下你自己。",
		},
	}

	fmt.Println("\n--- Testing Generation (Sync) ---")
	resp, err := gen.Call(context.Background(), req)
	if err != nil {
		fmt.Printf("Generation failed: %v\n", err)
	} else {
		fmt.Printf("Output: %s\n", resp.Output.Text)
		fmt.Printf("Usage: %d input, %d output\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}

	// Test 2: Streaming generation
	fmt.Println("\n--- Testing Generation (Stream) ---")
	reqStream := dashscope.GenerationRequest{
		Model: dashscope.QwenTurbo,
		Input: dashscope.GenerationInput{
			Prompt: "请讲一个关于程序员的笑话。",
		},
		Parameters: &dashscope.GenerationParameters{
			ResultFormat:      "text",
			IncrementalOutput: false, // Delta
		},
	}

	ch, err := gen.CallStream(context.Background(), reqStream)
	if err != nil {
		fmt.Printf("Stream call failed: %v\n", err)
		return
	}

	for resp := range ch {
		if resp.StatusCode != 0 && resp.StatusCode != 200 {
			fmt.Printf("Error: %s\n", resp.Message)
			break
		}
		fmt.Printf("%s", resp.Output.Text)
	}
	fmt.Println("\nStream finished")
}
