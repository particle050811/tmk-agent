package transcript

import (
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

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
