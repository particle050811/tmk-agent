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
	"regexp"
	"strconv"
	"strings"

	"tmk-agent/internal/config"
)

//go:embed prompt.txt
var transcriptPromptTemplate string

var srtTimestampPattern = regexp.MustCompile(`^\d{2}:\d{2}:\d{2},\d{3} --> \d{2}:\d{2}:\d{2},\d{3}$`)

type Service struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

type Result struct {
	Subtitles string
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func New(cfg config.Config) *Service {
	return &Service{
		baseURL: compatibleBaseURL(cfg.BaseURL),
		apiKey:  cfg.APIKey,
		model:   cfg.TranscriptModel,
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
		"model": s.model,
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
		return Result{}, buildAPIError(resp.Status, respBody, s.model)
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

	subtitles, err := parseSRT(content)
	if err != nil {
		return Result{}, fmt.Errorf("decode response: invalid srt content: %w", err)
	}

	return Result{
		Subtitles: subtitles,
	}, nil
}

func buildAPIError(status string, respBody []byte, model string) error {
	rawBody := strings.TrimSpace(string(respBody))

	var apiErr apiErrorResponse
	if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Code != "" {
		if apiErr.Error.Code == "access_denied" || apiErr.Error.Type == "access_denied" {
			return fmt.Errorf("transcript request failed: status=%s model=%s code=%s message=%s; current DashScope account likely does not have access to this model, try another enabled Omni model via QWEN_TRANSCRIPT_MODEL", status, model, apiErr.Error.Code, apiErr.Error.Message)
		}
		return fmt.Errorf("transcript request failed: status=%s model=%s code=%s message=%s", status, model, apiErr.Error.Code, apiErr.Error.Message)
	}

	return fmt.Errorf("transcript request failed: status=%s model=%s body=%s", status, model, rawBody)
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

func parseSRT(content string) (string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimSpace(stripCodeFence(normalized))
	if normalized == "" {
		return "", fmt.Errorf("empty content")
	}

	blocks := strings.Split(normalized, "\n\n")
	cues := make([]string, 0, len(blocks))

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) < 3 {
			return "", fmt.Errorf("invalid cue %q", block)
		}

		index, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil {
			return "", fmt.Errorf("invalid cue index %q", lines[0])
		}
		if index != len(cues)+1 {
			return "", fmt.Errorf("unexpected cue index %d", index)
		}

		timestamp := strings.TrimSpace(lines[1])
		if !srtTimestampPattern.MatchString(timestamp) {
			return "", fmt.Errorf("invalid timestamp %q", timestamp)
		}

		textLines := make([]string, 0, len(lines)-2)
		for _, line := range lines[2:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			textLines = append(textLines, line)
		}
		if len(textLines) == 0 {
			return "", fmt.Errorf("empty subtitle text for cue %d", index)
		}

		cues = append(cues, fmt.Sprintf("%d\n%s\n%s", index, timestamp, strings.Join(textLines, "\n")))
	}

	if len(cues) == 0 {
		return "", fmt.Errorf("no subtitle cues found")
	}

	return strings.Join(cues, "\n\n") + "\n", nil
}

func stripCodeFence(content string) string {
	if !strings.HasPrefix(content, "```") {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return content
	}
	if strings.TrimSpace(lines[len(lines)-1]) != "```" {
		return content
	}

	return strings.Join(lines[1:len(lines)-1], "\n")
}
