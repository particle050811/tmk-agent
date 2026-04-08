package streaming

import "testing"

func TestChunkerPush(t *testing.T) {
	chunker := NewChunker(16000, 1, 100)

	first := make([]byte, 1000)
	if got := chunker.Push(first); len(got) != 0 {
		t.Fatalf("expected no chunks, got %d", len(got))
	}

	second := make([]byte, 2200)
	got := chunker.Push(second)
	if len(got) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(got))
	}
	if len(got[0]) != 3200 {
		t.Fatalf("chunk size = %d, want %d", len(got[0]), 3200)
	}
}
