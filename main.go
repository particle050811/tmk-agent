package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"tmk-agent/internal/audio"
	"tmk-agent/internal/config"
	"tmk-agent/internal/render"
	"tmk-agent/internal/streaming"
	"tmk-agent/internal/transcript"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usageError()
	}

	switch os.Args[1] {
	case "stream":
		return runStream(os.Args[2:])
	case "transcript":
		return runTranscript(os.Args[2:])
	default:
		return usageError()
	}
}

func runStream(args []string) error {
	fs := flag.NewFlagSet("stream", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	sourceLang := fs.String("source-lang", "zh", "source language code")
	targetLang := fs.String("target-lang", "en", "target language code")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	renderer := render.NewTerminal(os.Stdout, cfg.Debug)
	audioIn := make(chan []byte, cfg.AudioBufferFrames)

	mic, err := audio.StartMicrophone(ctx, cfg.SampleRate, cfg.Channels, cfg.AudioDevice, audioIn)
	if err != nil {
		return err
	}
	defer mic.Close()

	renderer.PrintStatus(fmt.Sprintf(
		"streaming from %s to %s with model %s",
		*sourceLang,
		*targetLang,
		cfg.Model,
	))
	renderer.PrintStatus(fmt.Sprintf(
		"capture device: %s (default=%t)",
		mic.SelectedDeviceName(),
		mic.SelectedIsDefault(),
	))
	if cfg.Debug {
		renderer.PrintStatus("available capture devices: " + strings.Join(mic.AvailableDevices(), ", "))
	}

	return streaming.Run(ctx, streaming.RunConfig{
		Realtime:          cfg.RealtimeConfig(*sourceLang, *targetLang),
		AudioIn:           audioIn,
		Renderer:          renderer,
		ChunkMillis:       cfg.ChunkMillis,
		SampleRate:        int(cfg.SampleRate),
		Channels:          int(cfg.Channels),
		Debug:             cfg.Debug,
		DebugAudioDir:     cfg.DebugAudioDir,
		DebugAudioSeconds: cfg.DebugAudioSeconds,
	})
}

func runTranscript(args []string) error {
	fs := flag.NewFlagSet("transcript", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	filePath := fs.String("file", "", "input audio file")
	output := fs.String("output", "", "output text file")
	sourceLang := fs.String("source-lang", "", "source language code")
	targetLang := fs.String("target-lang", "", "target language code")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *filePath == "" || *output == "" || *sourceLang == "" || *targetLang == "" {
		return errors.New("transcript requires --file, --output, --source-lang, and --target-lang")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	svc := transcript.New(cfg)
	result, err := svc.TranscribeFile(*filePath, *sourceLang, *targetLang)
	if err != nil {
		return err
	}

	outputDir := filepath.Dir(*output)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	content := fmt.Sprintf("Source Language: %s\nTarget Language: %s\n\nTranslation:\n%s\n", *sourceLang, *targetLang, result.Translation)
	if err := os.WriteFile(*output, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "transcript written to %s\n", *output)
	return nil
}

func usageError() error {
	return errors.New("usage: mini-tmk-agent <stream|transcript> [flags]")
}
