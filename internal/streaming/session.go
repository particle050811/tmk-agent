package streaming

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"tmk-agent/internal/config"
	"tmk-agent/internal/realtime"
)

type Renderer interface {
	PrintStatus(status string)
	PrintTargetDelta(text string)
	PrintTargetFinal(text string)
	PrintError(err error)
}

type RunConfig struct {
	Realtime          config.RealtimeConfig
	AudioIn           <-chan []byte
	Renderer          Renderer
	ChunkMillis       int
	SampleRate        int
	Channels          int
	Debug             bool
	DebugAudioDir     string
	DebugAudioSeconds int
}

func Run(ctx context.Context, cfg RunConfig) error {
	backoff := time.Second

	for {
		err := runConnected(ctx, cfg)
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}

		cfg.Renderer.PrintError(err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		if backoff < 5*time.Second {
			backoff *= 2
		}
	}
}

func runConnected(ctx context.Context, cfg RunConfig) error {
	client, err := realtime.Dial(ctx, cfg.Realtime)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.SendSessionUpdate(ctx, cfg.Realtime); err != nil {
		return err
	}

	cfg.Renderer.PrintStatus("realtime session connected")
	if cfg.Debug {
		cfg.Renderer.PrintStatus("waiting for microphone PCM")
	}
	var recorder *DebugAudioRecorder
	if cfg.DebugAudioDir != "" {
		recorder, err = NewDebugAudioRecorder(cfg.DebugAudioDir, cfg.SampleRate, cfg.Channels, cfg.DebugAudioSeconds)
		if err != nil {
			return fmt.Errorf("init debug audio recorder: %w", err)
		}
		defer func() {
			if closeErr := recorder.Close(); closeErr != nil {
				cfg.Renderer.PrintError(fmt.Errorf("close debug audio recorder: %w", closeErr))
			}
		}()
		cfg.Renderer.PrintStatus(fmt.Sprintf("debug audio save enabled: %s (%ds per file)", cfg.DebugAudioDir, cfg.DebugAudioSeconds))
	}
	chunker := NewChunker(cfg.SampleRate, cfg.Channels, cfg.ChunkMillis)
	lastEventAt := time.Now()
	receivedPCM := false
	silentCallbacks := 0
	silentWarningShown := false
	sentChunks := 0
	idleTicker := time.NewTicker(5 * time.Second)
	defer idleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-client.Errs():
			if err == nil {
				return nil
			}
			return fmt.Errorf("realtime read loop: %w", err)
		case event, ok := <-client.Events():
			if !ok {
				return errors.New("realtime event stream closed")
			}
			lastEventAt = time.Now()
			if cfg.Debug {
				cfg.Renderer.PrintStatus("received event: " + event.Type)
			}
			handleEvent(cfg.Renderer, event)
		case pcm, ok := <-cfg.AudioIn:
			if !ok {
				return errors.New("audio input stream closed")
			}
			if recorder != nil {
				path, err := recorder.Write(pcm)
				if err != nil {
					return fmt.Errorf("write debug audio: %w", err)
				}
				if path != "" {
					cfg.Renderer.PrintStatus("debug audio saved: " + path)
				}
			}
			if cfg.Debug && !receivedPCM {
				receivedPCM = true
				cfg.Renderer.PrintStatus(fmt.Sprintf("microphone PCM received: %d bytes per callback", len(pcm)))
			}
			if pcmPeak16(pcm) == 0 {
				silentCallbacks++
				if cfg.Debug && !silentWarningShown && silentCallbacks >= 20 {
					silentWarningShown = true
					cfg.Renderer.PrintStatus("microphone callback is active but PCM is all zeros; check OS microphone privacy, input mute, or set TMK_AUDIO_DEVICE to another capture device")
				}
			} else {
				silentCallbacks = 0
			}
			for _, chunk := range chunker.Push(pcm) {
				if err := client.AppendAudio(ctx, chunk); err != nil {
					return fmt.Errorf("append audio: %w", err)
				}
				sentChunks++
				if cfg.Debug && sentChunks == 1 {
					cfg.Renderer.PrintStatus(fmt.Sprintf("first audio chunk sent: %d bytes", len(chunk)))
				}
			}
		case <-idleTicker.C:
			if !cfg.Debug {
				continue
			}
			if !receivedPCM {
				cfg.Renderer.PrintStatus("still no microphone PCM; check OS microphone access or runtime environment")
				continue
			}
			if time.Since(lastEventAt) >= 5*time.Second {
				cfg.Renderer.PrintStatus("audio is being captured, but no server event yet; speak louder or check server VAD/model behavior")
			}
		}
	}
}

func pcmPeak16(pcm []byte) int {
	peak := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int(int16(binary.LittleEndian.Uint16(pcm[i:])))
		if v < 0 {
			v = -v
		}
		if v > peak {
			peak = v
		}
	}
	return peak
}

func handleEvent(renderer Renderer, event realtime.Event) {
	switch event.Type {
	case "response.text.delta":
		if event.Delta != "" {
			renderer.PrintTargetDelta(event.Delta)
		}
	case "response.text.done":
		if event.Text != "" {
			renderer.PrintTargetFinal(event.Text)
		}
	case "error":
		renderer.PrintError(errors.New(event.Error))
	}
}
