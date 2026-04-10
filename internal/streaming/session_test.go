package streaming

import (
	"testing"

	"tmk-agent/internal/realtime"
)

type stubRenderer struct {
	deltas []string
	finals []string
	errs   []error
}

func (s *stubRenderer) PrintStatus(string) {}

func (s *stubRenderer) PrintTargetDelta(text string) {
	s.deltas = append(s.deltas, text)
}

func (s *stubRenderer) PrintTargetFinal(text string) {
	s.finals = append(s.finals, text)
}

func (s *stubRenderer) PrintError(err error) {
	s.errs = append(s.errs, err)
}

func TestHandleEvent(t *testing.T) {
	tests := []struct {
		name       string
		event      realtime.Event
		wantDeltas []string
		wantFinals []string
		wantErr    string
	}{
		{
			name:       "text delta",
			event:      realtime.Event{Type: "response.text.delta", Delta: "hello"},
			wantDeltas: []string{"hello"},
		},
		{
			name:       "audio transcript delta",
			event:      realtime.Event{Type: "response.audio_transcript.delta", Delta: "bonjour"},
			wantDeltas: []string{"bonjour"},
		},
		{
			name:       "text done",
			event:      realtime.Event{Type: "response.text.done", Text: "hello world"},
			wantFinals: []string{"hello world"},
		},
		{
			name:       "audio transcript done",
			event:      realtime.Event{Type: "response.audio_transcript.done", Text: "hola mundo"},
			wantFinals: []string{"hola mundo"},
		},
		{
			name:    "error",
			event:   realtime.Event{Type: "error", Error: "boom"},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := &stubRenderer{}

			handleEvent(renderer, tt.event)

			if len(renderer.deltas) != len(tt.wantDeltas) {
				t.Fatalf("delta count = %d, want %d", len(renderer.deltas), len(tt.wantDeltas))
			}
			for i := range tt.wantDeltas {
				if renderer.deltas[i] != tt.wantDeltas[i] {
					t.Fatalf("delta[%d] = %q, want %q", i, renderer.deltas[i], tt.wantDeltas[i])
				}
			}

			if len(renderer.finals) != len(tt.wantFinals) {
				t.Fatalf("final count = %d, want %d", len(renderer.finals), len(tt.wantFinals))
			}
			for i := range tt.wantFinals {
				if renderer.finals[i] != tt.wantFinals[i] {
					t.Fatalf("final[%d] = %q, want %q", i, renderer.finals[i], tt.wantFinals[i])
				}
			}

			switch {
			case tt.wantErr == "" && len(renderer.errs) != 0:
				t.Fatalf("unexpected errors: %v", renderer.errs)
			case tt.wantErr != "":
				if len(renderer.errs) != 1 {
					t.Fatalf("error count = %d, want 1", len(renderer.errs))
				}
				if renderer.errs[0].Error() != tt.wantErr {
					t.Fatalf("error = %q, want %q", renderer.errs[0], tt.wantErr)
				}
			}
		})
	}
}
