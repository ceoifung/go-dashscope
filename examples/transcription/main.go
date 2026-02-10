package main

import (
	"context"
	"fmt"
	"github.com/ceoifung/go-dashcope/dashscope"
	"os"
)

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set DASHSCOPE_API_KEY environment variable")
		return
	}

	transcription := dashscope.NewTranscription(apiKey)

	req := dashscope.TranscriptionRequest{
		Model: dashscope.ParaformerV1,
		Input: dashscope.TranscriptionInput{
			FileURLs: []string{"https://dashscope.oss-cn-beijing.aliyuncs.com/samples/audio/paraformer/hello_world.wav"},
		},
	}

	fmt.Println("\n--- Testing Transcription ---")
	resp, err := transcription.Call(context.Background(), req)
	if err != nil {
		fmt.Printf("Transcription failed: %v\n", err)
		return
	}

	fmt.Printf("Task Status: %s\n", resp.Output.TaskStatus)
	fmt.Printf("Results: %s\n", string(resp.Output.Results))
}
