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

	mm := dashscope.NewMultiModalConversation(apiKey)

	req := dashscope.MultiModalConversationRequest{
		Model: dashscope.QwenVLChatV1Plus,
		Input: dashscope.MultiModalConversationInput{
			Messages: []dashscope.MultiModalMessage{
				{
					Role: "user",
					Content: []dashscope.MultiModalContentItem{
						{Image: "https://dashscope.oss-cn-beijing.aliyuncs.com/images/dog_and_girl.jpeg"},
						{Text: "这是什么？"},
					},
				},
			},
		},
	}

	fmt.Println("\n--- Testing MultiModal Conversation ---")
	resp, err := mm.Call(context.Background(), req)
	if err != nil {
		fmt.Printf("MultiModal Conversation failed: %v\n", err)
		return
	}

	fmt.Printf("Request ID: %s\n", resp.RequestID)
	if len(resp.Output.Choices) > 0 {
		content := resp.Output.Choices[0].Message.Content
		for _, item := range content {
			if item.Text != "" {
				fmt.Printf("Response: %s\n", item.Text)
			}
		}
	}
	fmt.Printf("Usage: %d images, %d input tokens, %d output tokens\n", resp.Usage.ImageCount, resp.Usage.InputTokens, resp.Usage.OutputTokens)
}
