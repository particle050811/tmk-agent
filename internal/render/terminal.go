package render

import (
	"fmt"
	"io"
	"sync"
)

type Terminal struct {
	out   io.Writer
	debug bool
	mu    sync.Mutex
}

func NewTerminal(out io.Writer, debug bool) *Terminal {
	return &Terminal{out: out, debug: debug}
}

func (t *Terminal) PrintStatus(status string) {
	if !t.debug {
		return
	}
	t.printf("[status] %s\n", status)
}

func (t *Terminal) PrintTargetDelta(text string) {
	if !t.debug {
		return
	}
	t.printf("[target.partial] %s\n", text)
}

func (t *Terminal) PrintTargetFinal(text string) {
	t.printf("[target.final] %s\n", text)
}

func (t *Terminal) PrintError(err error) {
	t.printf("[error] %v\n", err)
}

func (t *Terminal) printf(format string, args ...any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, _ = fmt.Fprintf(t.out, format, args...)
}
