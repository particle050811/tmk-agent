package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"

	"tmk-agent/internal/config"
)

const transcriptionModel = "gummy-realtime-v1"

type Client struct {
	conn *websocket.Conn

	events chan Event
	errs   chan error

	readCtx    context.Context
	cancelRead context.CancelFunc
	closeOnce  sync.Once
}

func Dial(ctx context.Context, cfg config.RealtimeConfig) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, cfg.URL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization":              []string{"Bearer " + cfg.APIKey},
			"X-DashScope-DataInspection": []string{"disable"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dial realtime websocket: %w", err)
	}

	c := &Client{
		conn:   conn,
		events: make(chan Event, 128),
		errs:   make(chan error, 1),
	}
	c.readCtx, c.cancelRead = context.WithCancel(context.Background())

	go c.readLoop(c.readCtx)

	return c, nil
}

func (c *Client) Events() <-chan Event {
	return c.events
}

func (c *Client) Errs() <-chan error {
	return c.errs
}

func (c *Client) SendSessionUpdate(ctx context.Context, cfg config.RealtimeConfig) error {
	return c.writeJSON(ctx, SessionUpdateEvent{
		Type: "session.update",
		Session: SessionPayload{
			Modalities:       []string{"text"},
			InputAudioFormat: "pcm",
			InputAudioTranscription: &InputAudioTranscription{
				Model: transcriptionModel,
			},
			TurnDetection: &TurnDetection{
				Type:            "server_vad",
				Threshold:       0.5,
				PrefixPadding:   300,
				SilenceDuration: 500,
				CreateResponse:  true,
			},
			Instructions: buildInstructions(cfg.SourceLang, cfg.TargetLang),
		},
	})
}

func (c *Client) AppendAudio(ctx context.Context, pcm []byte) error {
	return c.writeJSON(ctx, InputAudioAppendEvent{
		Type:  "input_audio_buffer.append",
		Audio: base64.StdEncoding.EncodeToString(pcm),
	})
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.cancelRead != nil {
			c.cancelRead()
		}
		_ = c.conn.Close(websocket.StatusNormalClosure, "closing")
	})
}

func (c *Client) writeJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal realtime event: %w", err)
	}
	if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("write realtime event: %w", err)
	}
	return nil
}

func (c *Client) readLoop(ctx context.Context) {
	defer close(c.events)
	defer close(c.errs)

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}

			select {
			case c.errs <- err:
			default:
			}
			return
		}

		event, err := parseEvent(data)
		if err != nil {
			select {
			case c.errs <- err:
			default:
			}
			continue
		}

		select {
		case c.events <- event:
		case <-ctx.Done():
			return
		}
	}
}

func buildInstructions(sourceLang, targetLang string) string {
	return fmt.Sprintf(
		"Listen to the user's spoken %s input, transcribe it accurately, and respond only with the translated %s text. Keep punctuation natural and do not add commentary.",
		sourceLang,
		targetLang,
	)
}

func parseEvent(data []byte) (Event, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Event{}, fmt.Errorf("decode event: %w", err)
	}

	event := Event{Raw: append([]byte(nil), data...)}
	if v, ok := raw["type"]; ok {
		if err := json.Unmarshal(v, &event.Type); err != nil {
			return Event{}, fmt.Errorf("decode event type: %w", err)
		}
	}

	switch event.Type {
	case "conversation.item.input_audio_transcription.completed":
		var payload struct {
			ItemID         string `json:"item_id"`
			Transcript     string `json:"transcript"`
			ConversationID string `json:"conversation_id"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return Event{}, err
		}
		event.ID = payload.ItemID
		event.Text = payload.Transcript
	case "response.text.delta":
		var payload struct {
			ResponseID string `json:"response_id"`
			Delta      string `json:"delta"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return Event{}, err
		}
		event.ResponseID = payload.ResponseID
		event.Delta = payload.Delta
	case "response.text.done":
		var payload struct {
			ResponseID string `json:"response_id"`
			Text       string `json:"text"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return Event{}, err
		}
		event.ResponseID = payload.ResponseID
		event.Text = payload.Text
	case "error":
		var payload struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return Event{}, err
		}
		event.Error = payload.Error.Message
	}

	return event, nil
}
