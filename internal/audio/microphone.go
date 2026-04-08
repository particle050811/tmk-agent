package audio

import (
	"context"
	"fmt"
	"sync"

	"github.com/gen2brain/malgo"
)

type Microphone struct {
	ctx    *malgo.AllocatedContext
	device *malgo.Device

	closeOnce sync.Once
}

func StartMicrophone(ctx context.Context, sampleRate uint32, channels uint32, out chan<- []byte) (*Microphone, error) {
	maCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("init malgo context: %w", err)
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = channels
	cfg.SampleRate = sampleRate
	cfg.Alsa.NoMMap = 1

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, inputSamples []byte, _ uint32) {
			if len(inputSamples) == 0 {
				return
			}

			buf := make([]byte, len(inputSamples))
			copy(buf, inputSamples)

			select {
			case out <- buf:
			default:
			}
		},
	}

	device, err := malgo.InitDevice(maCtx.Context, cfg, callbacks)
	if err != nil {
		_ = maCtx.Uninit()
		maCtx.Free()
		return nil, fmt.Errorf("init capture device: %w", err)
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		_ = maCtx.Uninit()
		maCtx.Free()
		return nil, fmt.Errorf("start capture device: %w", err)
	}

	m := &Microphone{
		ctx:    maCtx,
		device: device,
	}

	go func() {
		<-ctx.Done()
		m.Close()
	}()

	return m, nil
}

func (m *Microphone) Close() {
	m.closeOnce.Do(func() {
		if m.device != nil {
			m.device.Uninit()
		}
		if m.ctx != nil {
			_ = m.ctx.Uninit()
			m.ctx.Free()
		}
	})
}
