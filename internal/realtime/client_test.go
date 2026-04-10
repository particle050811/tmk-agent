package realtime

import "testing"

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Event
	}{
		{
			name: "response delta",
			in:   `{"type":"response.text.delta","response_id":"resp-1","delta":"hello"}`,
			want: Event{Type: "response.text.delta", ResponseID: "resp-1", Delta: "hello"},
		},
		{
			name: "response done",
			in:   `{"type":"response.text.done","response_id":"resp-1","text":"hello world"}`,
			want: Event{Type: "response.text.done", ResponseID: "resp-1", Text: "hello world"},
		},
		{
			name: "response audio transcript delta",
			in:   `{"type":"response.audio_transcript.delta","response_id":"resp-1","delta":"hello"}`,
			want: Event{Type: "response.audio_transcript.delta", ResponseID: "resp-1", Delta: "hello"},
		},
		{
			name: "response audio transcript done",
			in:   `{"type":"response.audio_transcript.done","response_id":"resp-1","transcript":"hello world"}`,
			want: Event{Type: "response.audio_transcript.done", ResponseID: "resp-1", Text: "hello world"},
		},
		{
			name: "response audio delta",
			in:   `{"type":"response.audio.delta","response_id":"resp-1","delta":"aGVsbG8="}`,
			want: Event{Type: "response.audio.delta", ResponseID: "resp-1", Audio: []byte("hello")},
		},
		{
			name: "response audio done",
			in:   `{"type":"response.audio.done","response_id":"resp-1"}`,
			want: Event{Type: "response.audio.done", ResponseID: "resp-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEvent([]byte(tt.in))
			if err != nil {
				t.Fatalf("parseEvent() error = %v", err)
			}

			if got.Type != tt.want.Type || got.ID != tt.want.ID || got.Text != tt.want.Text || got.Delta != tt.want.Delta || got.ResponseID != tt.want.ResponseID || string(got.Audio) != string(tt.want.Audio) {
				t.Fatalf("parseEvent() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
