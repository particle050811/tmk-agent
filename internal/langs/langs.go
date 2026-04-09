package langs

import (
	"fmt"
	"strings"
)

var supported = map[string]string{
	"zh":       "zh",
	"zh-cn":    "zh",
	"zh-hans":  "zh",
	"cn":       "zh",
	"chinese":  "zh",
	"en":       "en",
	"en-us":    "en",
	"en-gb":    "en",
	"english":  "en",
	"es":       "es",
	"es-es":    "es",
	"spanish":  "es",
	"ja":       "ja",
	"ja-jp":    "ja",
	"jp":       "ja",
	"japanese": "ja",
}

var displayNames = map[string]string{
	"zh": "中文",
	"en": "英文",
	"es": "西班牙语",
	"ja": "日语",
}

func Normalize(v string) (string, error) {
	key := strings.TrimSpace(strings.ToLower(v))
	if key == "" {
		return "", fmt.Errorf("language is required; supported languages: zh, en, es, ja")
	}

	code, ok := supported[key]
	if !ok {
		return "", fmt.Errorf("unsupported language %q; supported languages: zh, en, es, ja", v)
	}

	return code, nil
}

func DisplayName(code string) string {
	if name, ok := displayNames[code]; ok {
		return name
	}
	return code
}
