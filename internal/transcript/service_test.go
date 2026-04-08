package transcript

import "testing"

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

func TestParseTranslation(t *testing.T) {
	content := "Translation:\nHello, world"

	translation := parseTranslation(content)
	if translation != "Hello, world" {
		t.Fatalf("translation = %q", translation)
	}
}
