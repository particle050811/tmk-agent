package realtime

import "testing"

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Event
	}{
		{
			name: "transcription completed",
			in:   `{"type":"conversation.item.input_audio_transcription.completed","item_id":"item-1","transcript":"你好"}`,
			want: Event{Type: "conversation.item.input_audio_transcription.completed", ID: "item-1", Text: "你好"},
		},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEvent([]byte(tt.in))
			if err != nil {
				t.Fatalf("parseEvent() error = %v", err)
			}

			if got.Type != tt.want.Type || got.ID != tt.want.ID || got.Text != tt.want.Text || got.Delta != tt.want.Delta || got.ResponseID != tt.want.ResponseID {
				t.Fatalf("parseEvent() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
