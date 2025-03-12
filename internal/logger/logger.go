package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

type handler struct {
	mu  *sync.Mutex
	out io.Writer
}

func newHandler() *handler {
	return &handler{mu: &sync.Mutex{}, out: os.Stdout}
}

func (*handler) Enabled(context.Context, slog.Level) bool { return true }

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)

	var l string
	switch r.Level {
	case slog.LevelDebug:
		l = cyan(bold(r.Level.String()))

	case slog.LevelInfo:
		l = blue(bold(r.Level.String()))

	case slog.LevelWarn:
		l = yellow(bold(r.Level.String()))

	case slog.LevelError:
		l = red(bold(r.Level.String()))
	}
	buf = append(buf, []byte(l)...)
	buf = append(buf, ' ')
	buf = append(buf, []byte(r.Message)...)
	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.out.Write(buf)
	return err
}

func (h *handler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *handler) WithGroup(string) slog.Handler      { return h }

var logger *slog.Logger = slog.New(newHandler())

func Debugf(format string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	r := slog.NewRecord(time.Time{}, slog.LevelDebug, fmt.Sprintf(format, args...), pcs[0])
	_ = logger.Handler().Handle(context.Background(), r)
}

func Infof(format string, args ...any) {
	// if !logger.Enabled(context.Background(), slog.LevelInfo) {
	// 	return
	// }
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	r := slog.NewRecord(time.Time{}, slog.LevelInfo, fmt.Sprintf(format, args...), pcs[0])
	_ = logger.Handler().Handle(context.Background(), r)
}

func Warnf(format string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	r := slog.NewRecord(time.Time{}, slog.LevelWarn, fmt.Sprintf(format, args...), pcs[0])
	_ = logger.Handler().Handle(context.Background(), r)
}

func Errorf(format string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	r := slog.NewRecord(time.Time{}, slog.LevelError, fmt.Sprintf(format, args...), pcs[0])
	_ = logger.Handler().Handle(context.Background(), r)
}
