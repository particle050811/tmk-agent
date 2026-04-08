package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tmk-agent/internal/audio"
	"tmk-agent/internal/config"
	"tmk-agent/internal/render"
	"tmk-agent/internal/streaming"
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

	renderer := render.NewTerminal(os.Stdout)
	audioIn := make(chan []byte, cfg.AudioBufferFrames)

	mic, err := audio.StartMicrophone(ctx, cfg.SampleRate, cfg.Channels, audioIn)
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

	return streaming.Run(ctx, streaming.RunConfig{
		Realtime:    cfg.RealtimeConfig(*sourceLang, *targetLang),
		AudioIn:     audioIn,
		Renderer:    renderer,
		ChunkMillis: cfg.ChunkMillis,
		SampleRate:  int(cfg.SampleRate),
		Channels:    int(cfg.Channels),
		Debug:       cfg.Debug,
	})
}

func runTranscript(args []string) error {
	fs := flag.NewFlagSet("transcript", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	filePath := fs.String("file", "", "input audio file")
	output := fs.String("output", "", "output text file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *filePath == "" || *output == "" {
		return errors.New("transcript requires --file and --output")
	}

	return errors.New("transcript mode is not implemented yet")
}

func usageError() error {
	return errors.New("usage: mini-tmk-agent <stream|transcript> [flags]")
}
