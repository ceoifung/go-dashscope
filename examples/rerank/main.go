/*
 * @Author: Ceoifung
 * @Date: 2026-02-10 19:36:23
 * @LastEditors: Ceoifung
 * @LastEditTime: 2026-02-10 19:36:27
 * @Description: XiaoRGEEK All Rights Reserved. Powered By Ceoifung
 */
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

	rerank := dashscope.NewTextReRank(apiKey)

	req := dashscope.TextReRankRequest{
		Model: dashscope.GteRerank,
		Input: dashscope.TextReRankInput{
			Query: "什么是人工智能？",
			Documents: []string{
				"人工智能（Artificial Intelligence），英文缩写为AI。",
				"今天天气真好。",
				"深度学习是机器学习的一个子集。",
			},
		},
		Parameters: &dashscope.TextReRankParameters{
			ReturnDocuments: true,
			TopN:            2,
		},
	}

	fmt.Println("\n--- Testing Text ReRank ---")
	resp, err := rerank.Call(context.Background(), req)
	if err != nil {
		fmt.Printf("ReRank failed: %v\n", err)
		return
	}

	fmt.Printf("Request ID: %s\n", resp.RequestID)
	for _, res := range resp.Output.Results {
		fmt.Printf("Index: %d, Score: %.4f, Doc: %v\n", res.Index, res.RelevanceScore, res.Document)
	}
}
