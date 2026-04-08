package streaming

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDebugAudioRecorderWritesFixedSegmentsAndFlushesRemainder(t *testing.T) {
	dir := t.TempDir()
	recorder, err := NewDebugAudioRecorder(dir, 16000, 1, 1)
	if err != nil {
		t.Fatalf("NewDebugAudioRecorder() error = %v", err)
	}

	firstPath, err := recorder.Write(make([]byte, 32000))
	if err != nil {
		t.Fatalf("Write() first error = %v", err)
	}
	if firstPath == "" {
		t.Fatal("Write() first path is empty")
	}

	secondPath, err := recorder.Write(make([]byte, 16000))
	if err != nil {
		t.Fatalf("Write() second error = %v", err)
	}
	if secondPath != "" {
		t.Fatalf("Write() second path = %q, want empty", secondPath)
	}

	if err := recorder.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.wav"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("wav files = %d, want 2", len(files))
	}

	assertWAVFileSize(t, files[0], 44+32000)
	assertWAVFileSize(t, files[1], 44+16000)
}

func assertWAVFileSize(t *testing.T, path string, want int) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.Size() != int64(want) {
		t.Fatalf("file size for %q = %d, want %d", path, info.Size(), want)
	}
}
