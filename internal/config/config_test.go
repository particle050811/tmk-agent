package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "test-key")
	t.Setenv("QWEN_REALTIME_BASE_URL", "wss://example.com/realtime")
	t.Setenv("QWEN_REALTIME_MODEL", "test-model")
	t.Setenv("QWEN_TRANSCRIPT_MODEL", "test-transcript-model")
	t.Setenv("TMK_SAMPLE_RATE", "24000")
	t.Setenv("TMK_CHANNELS", "2")
	t.Setenv("TMK_CHUNK_MILLIS", "100")
	t.Setenv("TMK_AUDIO_DEVICE", "Realtek")
	t.Setenv("TMK_DEBUG", "true")
	t.Setenv("TMK_DEBUG_AUDIO_DIR", "/tmp/tmk-audio")
	t.Setenv("TMK_DEBUG_AUDIO_SECONDS", "8")
	t.Setenv("TMK_OUTPUT_AUDIO_DIR", "/tmp/tmk-output")
	t.Setenv("TMK_OUTPUT_VOICE", "Jennifer")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.APIKey != "test-key" {
		t.Fatalf("APIKey = %q", cfg.APIKey)
	}
	if cfg.BaseURL != "wss://example.com/realtime" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.Model != "test-model" {
		t.Fatalf("Model = %q", cfg.Model)
	}
	if cfg.TranscriptModel != "test-transcript-model" {
		t.Fatalf("TranscriptModel = %q", cfg.TranscriptModel)
	}
	if cfg.SampleRate != 24000 {
		t.Fatalf("SampleRate = %d", cfg.SampleRate)
	}
	if cfg.Channels != 2 {
		t.Fatalf("Channels = %d", cfg.Channels)
	}
	if cfg.ChunkMillis != 100 {
		t.Fatalf("ChunkMillis = %d", cfg.ChunkMillis)
	}
	if cfg.AudioDevice != "Realtek" {
		t.Fatalf("AudioDevice = %q", cfg.AudioDevice)
	}
	if !cfg.Debug {
		t.Fatal("Debug = false, want true")
	}
	if cfg.DebugAudioDir != "/tmp/tmk-audio" {
		t.Fatalf("DebugAudioDir = %q", cfg.DebugAudioDir)
	}
	if cfg.DebugAudioSeconds != 8 {
		t.Fatalf("DebugAudioSeconds = %d", cfg.DebugAudioSeconds)
	}
	if cfg.OutputAudioDir != "/tmp/tmk-output" {
		t.Fatalf("OutputAudioDir = %q", cfg.OutputAudioDir)
	}
	if cfg.OutputVoice != "Jennifer" {
		t.Fatalf("OutputVoice = %q", cfg.OutputVoice)
	}
}

func TestLoadDefaultsOutputAudioDir(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "test-key")
	_ = os.Unsetenv("TMK_OUTPUT_AUDIO_DIR")
	_ = os.Unsetenv("TMK_OUTPUT_VOICE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OutputAudioDir != "./tmp/output-audio" {
		t.Fatalf("OutputAudioDir = %q", cfg.OutputAudioDir)
	}
	if cfg.OutputVoice != "Cherry" {
		t.Fatalf("OutputVoice = %q", cfg.OutputVoice)
	}
}

func TestLoadRequiresAPIKey(t *testing.T) {
	_ = os.Unsetenv("DASHSCOPE_API_KEY")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when api key is missing")
	}
}

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("DASHSCOPE_API_KEY=from-dotenv\nQWEN_REALTIME_MODEL=qwen-custom\nQWEN_TRANSCRIPT_MODEL=qwen-transcript\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	_ = os.Unsetenv("DASHSCOPE_API_KEY")
	_ = os.Unsetenv("QWEN_REALTIME_MODEL")
	_ = os.Unsetenv("QWEN_TRANSCRIPT_MODEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.APIKey != "from-dotenv" {
		t.Fatalf("APIKey = %q", cfg.APIKey)
	}
	if cfg.Model != "qwen-custom" {
		t.Fatalf("Model = %q", cfg.Model)
	}
	if cfg.TranscriptModel != "qwen-transcript" {
		t.Fatalf("TranscriptModel = %q", cfg.TranscriptModel)
	}
}
