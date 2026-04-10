package transcript

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestAudioFormat(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "a.mp3", want: "mp3"},
		{path: "a.WAV", want: "wav"},
		{path: "a.pcm", want: "pcm"},
	}

	for _, tt := range tests {
		got, err := audioFormat(tt.path)
		if err != nil {
			t.Fatalf("audioFormat(%q) error = %v", tt.path, err)
		}
		if got != tt.want {
			t.Fatalf("audioFormat(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestCompatibleBaseURL(t *testing.T) {
	if got := compatibleBaseURL("wss://dashscope.aliyuncs.com/api-ws/v1/realtime"); got != "https://dashscope.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("compatibleBaseURL() = %q", got)
	}
	if got := compatibleBaseURL("wss://dashscope-intl.aliyuncs.com/api-ws/v1/realtime"); got != "https://dashscope-intl.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("compatibleBaseURL() = %q", got)
	}
}

func TestParseSRT(t *testing.T) {
	content := "```srt\n1\n00:00:00,000 --> 00:00:01,500\nHello, world\n\n2\n00:00:01,500 --> 00:00:03,000\nHow are you?\n```"

	subtitles, err := parseSRT(content)
	if err != nil {
		t.Fatalf("parseSRT() error = %v", err)
	}

	want := "1\n00:00:00,000 --> 00:00:01,500\nHello, world\n\n2\n00:00:01,500 --> 00:00:03,000\nHow are you?\n"
	if subtitles != want {
		t.Fatalf("subtitles = %q, want %q", subtitles, want)
	}
}

func TestParseSRTRejectsInvalidCueSequence(t *testing.T) {
	content := "1\n00:00:00,000 --> 00:00:01,500\nHello\n\n3\n00:00:01,500 --> 00:00:03,000\nWorld"

	if _, err := parseSRT(content); err == nil {
		t.Fatal("parseSRT() error = nil, want error")
	}
}

func TestBuildAPIErrorAccessDenied(t *testing.T) {
	err := buildAPIError("403 Forbidden", []byte(`{"error":{"message":"Access denied","type":"access_denied","code":"access_denied"}}`), "qwen3.5-omni-plus")
	if err == nil {
		t.Fatal("buildAPIError() error = nil")
	}
	if got := err.Error(); got == "" || !containsAll(got, []string{"403 Forbidden", "qwen3.5-omni-plus", "access_denied"}) {
		t.Fatalf("buildAPIError() = %q", got)
	}
}

func TestTranscriptRequestSeparatesSystemPromptFromUserAudio(t *testing.T) {
	prompt := buildPrompt("中文", "English")
	payload := map[string]any{
		"model": "test-model",
		"messages": []map[string]any{
			{
				"role":    "system",
				"content": prompt,
			},
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_audio",
						"input_audio": map[string]any{
							"data":   "data:;base64," + base64.StdEncoding.EncodeToString([]byte("audio")),
							"format": "mp3",
						},
					},
				},
			},
		},
		"modalities": []string{"text"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(decoded.Messages) != 2 {
		t.Fatalf("messages count = %d, want 2", len(decoded.Messages))
	}
	if decoded.Messages[0].Role != "system" {
		t.Fatalf("first message role = %q, want system", decoded.Messages[0].Role)
	}
	if decoded.Messages[0].Content != prompt {
		t.Fatalf("system content = %v, want %q", decoded.Messages[0].Content, prompt)
	}
	if decoded.Messages[1].Role != "user" {
		t.Fatalf("second message role = %q, want user", decoded.Messages[1].Role)
	}

	userContent, ok := decoded.Messages[1].Content.([]any)
	if !ok {
		t.Fatalf("user content type = %T, want []any", decoded.Messages[1].Content)
	}
	if len(userContent) != 1 {
		t.Fatalf("user content length = %d, want 1", len(userContent))
	}

	audioPart, ok := userContent[0].(map[string]any)
	if !ok {
		t.Fatalf("user content part type = %T, want map[string]any", userContent[0])
	}
	if audioPart["type"] != "input_audio" {
		t.Fatalf("user content part type field = %v, want input_audio", audioPart["type"])
	}
	if _, exists := audioPart["text"]; exists {
		t.Fatal("user content unexpectedly contains prompt text")
	}
}

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
