package dashscope

const (
	// Base URLs
	BaseWebsocketURL = "wss://dashscope.aliyuncs.com/api-ws/v1/inference"
	
	// Service URLs
	
	// Text Generation
	QwenGenerationURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	
	// Multimodal Generation (Qwen-VL)
	QwenVLGenerationURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	
	// Image Synthesis (Wanx)
	ImageSynthesisURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis"
	
	// Embeddings
	TextEmbeddingURL = "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding"
	
	// Natural Language Understanding (NLU)
	NLUUnderstandingURL = "https://dashscope.aliyuncs.com/api/v1/services/nlp/nlu/understanding"
	
	// ReRank
	TextReRankURL = "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank"
	
	// Audio - ASR (Transcription)
	ASRTranscriptionURL = "https://dashscope.aliyuncs.com/api/v1/services/audio/asr/transcription"
	
	// Tasks (Async)
	TaskBaseURL = "https://dashscope.aliyuncs.com/api/v1/tasks"
)
