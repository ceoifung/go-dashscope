/*
 * @Author: Ceoifung
 * @Date: 2026-02-10 17:58:49
 * @LastEditors: Ceoifung
 * @LastEditTime: 2026-02-10 19:09:57
 * @Description: XiaoRGEEK All Rights Reserved. Powered By Ceoifung
 */
package dashscope

const (
	TaskGroupAudio = "audio"
	TaskTTS        = "tts"
	FunctionTTS    = "SpeechSynthesizer"
)

type EventType string

const (
	EventTaskStarted     EventType = "task-started"
	EventResultGenerated EventType = "result-generated"
	EventTaskFinished    EventType = "task-finished"
	EventTaskFailed      EventType = "task-failed"
)

type ActionType string

const (
	ActionRunTask ActionType = "run-task"
)

// AudioFormat constants
const (
	AudioFormatWAV = "wav"
	AudioFormatPCM = "pcm"
	AudioFormatMP3 = "mp3"
)

// Role constants
const (
	RoleUser      = "user"
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleBot       = "bot"
)

// Message represents a message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
