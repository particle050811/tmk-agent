package transcript

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tmk-agent/internal/config"
)

const transcriptModel = "qwen3-omni-flash"

//go:embed prompt.txt
var transcriptPromptTemplate string

type Service struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type Result struct {
	Translation string
}

func New(cfg config.Config) *Service {
	return &Service{
		baseURL: compatibleBaseURL(cfg.BaseURL),
		apiKey:  cfg.APIKey,
		client:  &http.Client{},
	}
}

func (s *Service) TranscribeFile(path string, sourceLang string, targetLang string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read audio file: %w", err)
	}

	format, err := audioFormat(path)
	if err != nil {
		return Result{}, err
	}

	payload := map[string]any{
		"model": transcriptModel,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_audio",
						"input_audio": map[string]any{
							"data":   "data:;base64," + base64.StdEncoding.EncodeToString(data),
							"format": format,
						},
					},
					{
						"type": "text",
						"text": buildPrompt(sourceLang, targetLang),
					},
				},
			},
		},
		"modalities": []string{"text"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("transcript request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return Result{}, fmt.Errorf("decode response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return Result{}, fmt.Errorf("decode response: no choices returned")
	}

	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return Result{}, fmt.Errorf("decode response: empty transcript")
	}

	translation := parseTranslation(content)
	if translation == "" {
		return Result{
			Translation: content,
		}, nil
	}

	return Result{
		Translation: translation,
	}, nil
}

func compatibleBaseURL(realtimeBaseURL string) string {
	if strings.Contains(realtimeBaseURL, "dashscope-intl.aliyuncs.com") {
		return "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
	}
	return "https://dashscope.aliyuncs.com/compatible-mode/v1"
}

func audioFormat(path string) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	switch ext {
	case "mp3", "wav", "pcm":
		return ext, nil
	default:
		return "", fmt.Errorf("unsupported audio format %q, expected mp3, wav, or pcm", ext)
	}
}

func buildPrompt(sourceLang string, targetLang string) string {
	prompt := strings.ReplaceAll(transcriptPromptTemplate, "{{source_lang}}", sourceLang)
	prompt = strings.ReplaceAll(prompt, "{{target_lang}}", targetLang)
	return prompt
}

func parseTranslation(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.TrimSpace(sectionBody(normalized, "Translation:", ""))
}

func sectionBody(content string, start string, end string) string {
	startIdx := strings.Index(content, start)
	if startIdx == -1 {
		return ""
	}

	body := content[startIdx+len(start):]
	if end != "" {
		if endIdx := strings.Index(body, end); endIdx != -1 {
			body = body[:endIdx]
		}
	}

	return body
}
