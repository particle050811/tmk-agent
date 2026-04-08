package realtime

import "encoding/json"

type SessionUpdateEvent struct {
	Type    string         `json:"type"`
	Session SessionPayload `json:"session"`
}

type SessionPayload struct {
	Modalities       []string       `json:"modalities,omitempty"`
	Instructions     string         `json:"instructions,omitempty"`
	InputAudioFormat string         `json:"input_audio_format,omitempty"`
	TurnDetection    *TurnDetection `json:"turn_detection,omitempty"`
}

type TurnDetection struct {
	Type            string  `json:"type"`
	Threshold       float64 `json:"threshold,omitempty"`
	PrefixPadding   int     `json:"prefix_padding_ms,omitempty"`
	SilenceDuration int     `json:"silence_duration_ms,omitempty"`
	CreateResponse  bool    `json:"create_response,omitempty"`
}

type InputAudioAppendEvent struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

type RawEvent struct {
	Type string `json:"type"`
	Raw  []byte `json:"-"`
}

type Event struct {
	Type       string
	ID         string
	Text       string
	Delta      string
	ResponseID string
	Error      string
	Arguments  json.RawMessage
	Raw        []byte
}
