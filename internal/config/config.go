package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultBaseURL                  = "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
	defaultModel                    = "qwen3.5-omni-plus-realtime"
	defaultSampleRate        uint32 = 16000
	defaultChannels          uint32 = 1
	defaultChunkMillis              = 200
	defaultAudioBufferFrames        = 64
)

type Config struct {
	APIKey            string
	BaseURL           string
	Model             string
	SampleRate        uint32
	Channels          uint32
	ChunkMillis       int
	AudioBufferFrames int
	Debug             bool
}

type RealtimeConfig struct {
	URL         string
	APIKey      string
	Model       string
	SourceLang  string
	TargetLang  string
	SampleRate  uint32
	Channels    uint32
	ChunkMillis int
}

func Load() (Config, error) {
	loadDotEnv(".env")

	cfg := Config{
		APIKey:            os.Getenv("DASHSCOPE_API_KEY"),
		BaseURL:           getenv("QWEN_REALTIME_BASE_URL", defaultBaseURL),
		Model:             getenv("QWEN_REALTIME_MODEL", defaultModel),
		SampleRate:        getEnvUint32("TMK_SAMPLE_RATE", defaultSampleRate),
		Channels:          getEnvUint32("TMK_CHANNELS", defaultChannels),
		ChunkMillis:       getEnvInt("TMK_CHUNK_MILLIS", defaultChunkMillis),
		AudioBufferFrames: getEnvInt("TMK_AUDIO_BUFFER_FRAMES", defaultAudioBufferFrames),
		Debug:             getEnvBool("TMK_DEBUG", false),
	}

	if cfg.APIKey == "" {
		return Config{}, errors.New("DASHSCOPE_API_KEY is required")
	}
	if cfg.ChunkMillis <= 0 {
		return Config{}, errors.New("TMK_CHUNK_MILLIS must be > 0")
	}
	if cfg.AudioBufferFrames <= 0 {
		return Config{}, errors.New("TMK_AUDIO_BUFFER_FRAMES must be > 0")
	}

	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return Config{}, fmt.Errorf("parse QWEN_REALTIME_BASE_URL: %w", err)
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return Config{}, errors.New("QWEN_REALTIME_BASE_URL must use ws or wss")
	}

	return cfg, nil
}

func loadDotEnv(path string) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		_ = os.Setenv(key, value)
	}
}

func (c Config) RealtimeConfig(sourceLang, targetLang string) RealtimeConfig {
	base := strings.TrimRight(c.BaseURL, "/")
	return RealtimeConfig{
		URL:         fmt.Sprintf("%s?model=%s", base, url.QueryEscape(c.Model)),
		APIKey:      c.APIKey,
		Model:       c.Model,
		SourceLang:  normalizeLanguage(sourceLang),
		TargetLang:  normalizeLanguage(targetLang),
		SampleRate:  c.SampleRate,
		Channels:    c.Channels,
		ChunkMillis: c.ChunkMillis,
	}
}

func normalizeLanguage(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "auto"
	}
	return v
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvUint32(key string, fallback uint32) uint32 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return fallback
	}
	return uint32(n)
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}

	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
