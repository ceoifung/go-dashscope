/*
 * @Author: Ceoifung
 * @Date: 2026-02-10 19:29:57
 * @LastEditors: Ceoifung
 * @LastEditTime: 2026-02-10 19:57:42
 * @Description: XiaoRGEEK All Rights Reserved. Powered By Ceoifung
 */
// Package dashscope provides a Go SDK for Alibaba Cloud DashScope (ModelScope) API.
//
// It supports various services including:
// - Text Generation (Qwen/Tongyi Qianwen)
// - Image Synthesis (Wanx/Tongyi Wanxiang)
// - Audio Recognition (Paraformer)
// - Text-to-Speech (Sambert)
// - Multimodal Conversation (Qwen-VL)
// - Text Embeddings and Reranking
// - Natural Language Understanding (NLU)
//
// Basic usage:
//
//	gen := dashscope.NewGeneration("your-api-key")
//	resp, err := gen.Call(context.Background(), req)
//
// See individual client types for more details.
package dashscope
