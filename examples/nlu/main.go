package main

import (
	"context"
	"fmt"
	"go-dashcope/dashscope"
	"os"
)

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set DASHSCOPE_API_KEY environment variable")
		return
	}

	nlu := dashscope.NewUnderstanding(apiKey)

	req := dashscope.UnderstandingRequest{
		Model: dashscope.OpenNLUV1,
		Input: dashscope.UnderstandingInput{
			Sentence: "今天杭州的天气怎么样？",
			Labels:   "天气,美食,交通",
			Task:     "classification",
		},
	}

	fmt.Println("\n--- Testing Understanding (NLU) ---")
	resp, err := nlu.Call(context.Background(), req)
	if err != nil {
		fmt.Printf("Understanding failed: %v\n", err)
		return
	}

	fmt.Printf("Request ID: %s\n", resp.RequestID)
	fmt.Printf("Results: %s\n", string(resp.Output))
}
