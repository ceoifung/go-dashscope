package main

import (
	"context"
	"fmt"
	"github.com/ceoifung/go-dashscope/dashscope"
	"os"
)

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set DASHSCOPE_API_KEY environment variable")
		return
	}

	emb := dashscope.NewTextEmbedding(apiKey)

	req := dashscope.TextEmbeddingRequest{
		Model: dashscope.TextEmbeddingV1,
		Input: dashscope.TextEmbeddingInput{
			Texts: []string{"风急天高猿啸哀", "渚清沙白鸟飞回", "无边落木萧萧下", "不尽长江滚滚来"},
		},
		Parameters: &dashscope.TextEmbeddingParameters{
			TextType: "document",
		},
	}

	fmt.Println("\n--- Testing Text Embedding ---")
	resp, err := emb.Call(context.Background(), req)
	if err != nil {
		fmt.Printf("Text Embedding failed: %v\n", err)
		return
	}

	fmt.Printf("Request ID: %s\n", resp.RequestID)
	fmt.Printf("Total Tokens: %d\n", resp.Usage.TotalTokens)
	for i, res := range resp.Output.Embeddings {
		fmt.Printf("Text %d Embedding Length: %d\n", i, len(res.Embedding))
	}
}
