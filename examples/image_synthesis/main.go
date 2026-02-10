/*
 * @Author: Ceoifung
 * @Date: 2026-02-10 19:35:44
 * @LastEditors: Ceoifung
 * @LastEditTime: 2026-02-10 20:04:18
 * @Description: XiaoRGEEK All Rights Reserved. Powered By Ceoifung
 */
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ceoifung/go-dashscope/dashscope"
)

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set DASHSCOPE_API_KEY environment variable")
		return
	}

	imgSynth := dashscope.NewImageSynthesis(apiKey)

	req := dashscope.ImageSynthesisRequest{
		Model: dashscope.WanxV1,
		Input: dashscope.ImageSynthesisInput{
			Prompt: "一只戴着墨镜的猫，赛博朋克风格",
		},
		Parameters: &dashscope.ImageSynthesisParameters{
			Size:  "1024*1024",
			N:     1,
			Style: "<auto>",
		},
	}

	fmt.Println("\n--- Testing Image Synthesis ---")
	resp, err := imgSynth.Call(context.Background(), req)
	if err != nil {
		fmt.Printf("Image Synthesis failed: %v\n", err)
		return
	}

	fmt.Printf("Task Status: %s\n", resp.Output.TaskStatus)
	for i, res := range resp.Output.Results {
		fmt.Printf("Image %d URL: %s\n", i+1, res.URL)
	}
}
