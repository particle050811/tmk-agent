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
	t.Setenv("TMK_SAMPLE_RATE", "24000")
	t.Setenv("TMK_CHANNELS", "2")
	t.Setenv("TMK_CHUNK_MILLIS", "100")
	t.Setenv("TMK_DEBUG", "true")

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
	if cfg.SampleRate != 24000 {
		t.Fatalf("SampleRate = %d", cfg.SampleRate)
	}
	if cfg.Channels != 2 {
		t.Fatalf("Channels = %d", cfg.Channels)
	}
	if cfg.ChunkMillis != 100 {
		t.Fatalf("ChunkMillis = %d", cfg.ChunkMillis)
	}
	if !cfg.Debug {
		t.Fatal("Debug = false, want true")
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
	if err := os.WriteFile(envFile, []byte("DASHSCOPE_API_KEY=from-dotenv\nQWEN_REALTIME_MODEL=qwen-custom\n"), 0o644); err != nil {
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
}
