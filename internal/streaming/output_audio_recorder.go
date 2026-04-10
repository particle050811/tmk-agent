package streaming

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const realtimeOutputSampleRate = 24000

type OutputAudioRecorder struct {
	dir      string
	channels int

	mu     sync.Mutex
	seq    int
	buffer []byte
}

func NewOutputAudioRecorder(dir string, channels int) (*OutputAudioRecorder, error) {
	if dir == "" {
		return nil, fmt.Errorf("output audio dir is empty")
	}
	if channels <= 0 {
		return nil, fmt.Errorf("output audio channels must be > 0")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create output audio dir: %w", err)
	}

	return &OutputAudioRecorder{
		dir:      dir,
		channels: channels,
	}, nil
}

func (r *OutputAudioRecorder) Append(pcm []byte) {
	if len(pcm) == 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.buffer = append(r.buffer, pcm...)
}

func (r *OutputAudioRecorder) Flush(responseID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.buffer) == 0 {
		return "", nil
	}

	r.seq++
	name := fmt.Sprintf("%s-%03d.wav", sanitizeFileName(responseID), r.seq)
	path := filepath.Join(r.dir, name)
	if err := writePCM16WAV(path, r.buffer, realtimeOutputSampleRate, r.channels); err != nil {
		return "", err
	}

	r.buffer = nil
	return path, nil
}

func sanitizeFileName(s string) string {
	if s == "" {
		return fmt.Sprintf("response-%d", time.Now().UnixMilli())
	}

	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c)
		case c >= '0' && c <= '9':
			out = append(out, c)
		case c == '-', c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

func writePCM16WAV(path string, pcm []byte, sampleRate int, channels int) error {
	const bitsPerSample = 16
	const audioFormat = 1

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create wav parent dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create wav file: %w", err)
	}
	defer f.Close()

	blockAlign := uint16(channels * bitsPerSample / 8)
	byteRate := uint32(sampleRate) * uint32(blockAlign)
	dataSize := uint32(len(pcm))
	riffSize := 36 + dataSize

	if _, err := f.Write([]byte("RIFF")); err != nil {
		return fmt.Errorf("write riff header: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, riffSize); err != nil {
		return fmt.Errorf("write riff size: %w", err)
	}
	if _, err := f.Write([]byte("WAVEfmt ")); err != nil {
		return fmt.Errorf("write wave/fmt header: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil {
		return fmt.Errorf("write fmt chunk size: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(audioFormat)); err != nil {
		return fmt.Errorf("write audio format: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(channels)); err != nil {
		return fmt.Errorf("write channels: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return fmt.Errorf("write sample rate: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, byteRate); err != nil {
		return fmt.Errorf("write byte rate: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, blockAlign); err != nil {
		return fmt.Errorf("write block align: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(bitsPerSample)); err != nil {
		return fmt.Errorf("write bits per sample: %w", err)
	}
	if _, err := f.Write([]byte("data")); err != nil {
		return fmt.Errorf("write data header: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, dataSize); err != nil {
		return fmt.Errorf("write data size: %w", err)
	}
	if _, err := f.Write(pcm); err != nil {
		return fmt.Errorf("write pcm data: %w", err)
	}

	return nil
}
