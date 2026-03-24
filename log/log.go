package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"
)

// Tracef logs at LevelDebug with printf-style formatting.
func Tracef(format string, args ...any) {
	log(context.Background(), slog.LevelDebug, format, args...)
}

// Debugf logs at LevelDebug with printf-style formatting.
func Debugf(format string, args ...any) {
	log(context.Background(), slog.LevelDebug, format, args...)
}

// Infof logs at LevelInfo with printf-style formatting.
func Infof(format string, args ...any) {
	log(context.Background(), slog.LevelInfo, format, args...)
}

// Info logs at LevelInfo with printf-style formatting.
func Info(msg string) {
	log(context.Background(), slog.LevelInfo, msg)
}

// Warnf logs at LevelWarn with printf-style formatting.
func Warnf(format string, args ...any) {
	log(context.Background(), slog.LevelWarn, format, args...)
}

// Errorf logs at LevelError with printf-style formatting.
func Errorf(format string, args ...any) {
	log(context.Background(), slog.LevelError, format, args...)
}

// dispatch is the central internal orchestrator.
func log(ctx context.Context, level slog.Level, format string, args ...any) {
	logger := slog.Default()
	if !logger.Enabled(ctx, level) {
		return
	}

	// 1. Determine the call site.
	// Skip 3 frames:
	// [0] runtime.Callers
	// [1] dispatch (this function)
	// [2] Wrapper (e.g., Info or Infof)
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	pc := pcs[0]

	// 2. Handle formatting safely.
	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}

	// 3. Create the record with the captured PC.
	r := slog.NewRecord(time.Now(), level, msg, pc)
	_ = logger.Handler().Handle(ctx, r)
}
