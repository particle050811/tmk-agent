package streaming

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type DebugAudioRecorder struct {
	dir          string
	sampleRate   int
	channels     int
	segmentBytes int
	startedAt    time.Time
	segmentIndex int
	buf          []byte
}

func NewDebugAudioRecorder(dir string, sampleRate int, channels int, segmentSeconds int) (*DebugAudioRecorder, error) {
	if dir == "" {
		return nil, fmt.Errorf("debug audio dir is empty")
	}
	if sampleRate <= 0 || channels <= 0 || segmentSeconds <= 0 {
		return nil, fmt.Errorf("invalid recorder params: sampleRate=%d channels=%d seconds=%d", sampleRate, channels, segmentSeconds)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create debug audio dir: %w", err)
	}

	bytesPerSecond := sampleRate * channels * 2
	return &DebugAudioRecorder{
		dir:          dir,
		sampleRate:   sampleRate,
		channels:     channels,
		segmentBytes: bytesPerSecond * segmentSeconds,
		startedAt:    time.Now(),
	}, nil
}

func (r *DebugAudioRecorder) Write(pcm []byte) (string, error) {
	if len(pcm) == 0 {
		return "", nil
	}

	r.buf = append(r.buf, pcm...)
	if len(r.buf) < r.segmentBytes {
		return "", nil
	}

	return r.flushSegment(r.segmentBytes)
}

func (r *DebugAudioRecorder) Close() error {
	if len(r.buf) == 0 {
		return nil
	}
	_, err := r.flushSegment(len(r.buf))
	return err
}

func (r *DebugAudioRecorder) flushSegment(n int) (string, error) {
	if n <= 0 || len(r.buf) < n {
		return "", nil
	}

	payload := make([]byte, n)
	copy(payload, r.buf[:n])
	r.buf = r.buf[n:]

	name := fmt.Sprintf(
		"mic-%s-%03d.wav",
		r.startedAt.Format("20060102-150405"),
		r.segmentIndex,
	)
	r.segmentIndex++

	path := filepath.Join(r.dir, name)
	if err := writePCMAsWAV(path, payload, r.sampleRate, r.channels); err != nil {
		return "", err
	}
	return path, nil
}

func writePCMAsWAV(path string, pcm []byte, sampleRate int, channels int) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create wav file: %w", err)
	}
	defer f.Close()

	const (
		audioFormat   uint16 = 1
		bitsPerSample uint16 = 16
	)

	byteRate := uint32(sampleRate * channels * int(bitsPerSample/8))
	blockAlign := uint16(channels * int(bitsPerSample/8))
	dataSize := uint32(len(pcm))
	riffSize := 36 + dataSize

	if _, err := f.Write([]byte("RIFF")); err != nil {
		return fmt.Errorf("write wav RIFF: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, riffSize); err != nil {
		return fmt.Errorf("write wav size: %w", err)
	}
	if _, err := f.Write([]byte("WAVE")); err != nil {
		return fmt.Errorf("write wav WAVE: %w", err)
	}
	if _, err := f.Write([]byte("fmt ")); err != nil {
		return fmt.Errorf("write wav fmt: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil {
		return fmt.Errorf("write wav fmt size: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, audioFormat); err != nil {
		return fmt.Errorf("write wav audio format: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(channels)); err != nil {
		return fmt.Errorf("write wav channels: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return fmt.Errorf("write wav sample rate: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, byteRate); err != nil {
		return fmt.Errorf("write wav byte rate: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, blockAlign); err != nil {
		return fmt.Errorf("write wav block align: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, bitsPerSample); err != nil {
		return fmt.Errorf("write wav bits per sample: %w", err)
	}
	if _, err := f.Write([]byte("data")); err != nil {
		return fmt.Errorf("write wav data: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, dataSize); err != nil {
		return fmt.Errorf("write wav data size: %w", err)
	}
	if _, err := f.Write(pcm); err != nil {
		return fmt.Errorf("write wav payload: %w", err)
	}
	return nil
}
