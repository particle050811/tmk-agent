package audio

// #include <stdlib.h>
import "C"

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/gen2brain/malgo"
)

type Microphone struct {
	ctx    *malgo.AllocatedContext
	device *malgo.Device

	selectedDeviceName string
	selectedIsDefault  bool
	availableDevices   []string

	closeOnce sync.Once
}

func StartMicrophone(ctx context.Context, sampleRate uint32, channels uint32, deviceName string, out chan<- []byte) (*Microphone, error) {
	backends := []malgo.Backend(nil)
	if runtime.GOOS == "windows" {
		backends = []malgo.Backend{malgo.BackendWasapi}
	}

	maCtx, err := malgo.InitContext(backends, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("init malgo context: %w", err)
	}

	devices, err := maCtx.Context.Devices(malgo.Capture)
	if err != nil {
		_ = maCtx.Uninit()
		maCtx.Free()
		return nil, fmt.Errorf("list capture devices: %w", err)
	}

	availableDevices := make([]string, 0, len(devices))
	for _, dev := range devices {
		name := dev.Name()
		if dev.IsDefault != 0 {
			name += " [default]"
		}
		availableDevices = append(availableDevices, name)
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = channels
	cfg.SampleRate = sampleRate
	cfg.Alsa.NoMMap = 1

	selectedDeviceName := "system default"
	selectedIsDefault := true
	var releaseCaptureDeviceID func()

	if strings.TrimSpace(deviceName) != "" {
		selected, ok := findCaptureDevice(devices, deviceName)
		if !ok {
			_ = maCtx.Uninit()
			maCtx.Free()
			return nil, fmt.Errorf("capture device %q not found; available devices: %s", deviceName, strings.Join(availableDevices, ", "))
		}

		cfg.Capture.DeviceID = selected.ID.Pointer()
		releaseCaptureDeviceID = func() {
			C.free(cfg.Capture.DeviceID)
		}
		selectedDeviceName = selected.Name()
		selectedIsDefault = selected.IsDefault != 0
	} else {
		for _, dev := range devices {
			if dev.IsDefault != 0 {
				selectedDeviceName = dev.Name()
				break
			}
		}
	}

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
	if releaseCaptureDeviceID != nil {
		releaseCaptureDeviceID()
	}
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
		ctx:                maCtx,
		device:             device,
		selectedDeviceName: selectedDeviceName,
		selectedIsDefault:  selectedIsDefault,
		availableDevices:   availableDevices,
	}

	go func() {
		<-ctx.Done()
		m.Close()
	}()

	return m, nil
}

func findCaptureDevice(devices []malgo.DeviceInfo, needle string) (malgo.DeviceInfo, bool) {
	needle = strings.ToLower(strings.TrimSpace(needle))
	for _, dev := range devices {
		if strings.Contains(strings.ToLower(dev.Name()), needle) {
			return dev, true
		}
	}
	return malgo.DeviceInfo{}, false
}

func (m *Microphone) SelectedDeviceName() string {
	return m.selectedDeviceName
}

func (m *Microphone) SelectedIsDefault() bool {
	return m.selectedIsDefault
}

func (m *Microphone) AvailableDevices() []string {
	out := make([]string, len(m.availableDevices))
	copy(out, m.availableDevices)
	return out
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
