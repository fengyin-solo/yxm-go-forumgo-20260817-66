package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Level represents the logging severity level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is a structured key-value logger.
type Logger struct {
	out   io.Writer
	level Level
	ctx   map[string]any
	buf   *strings.Builder
	mu    sync.Mutex
}

// New creates a new Logger writing to out at the given level.
func New(out io.Writer, level Level) *Logger {
	buf := &strings.Builder{}
	return &Logger{
		out:   out,
		level: level,
		ctx:   make(map[string]any),
		buf:   buf,
	}
}

// Default returns a logger backed by os.Stdout at LevelInfo.
func Default() *Logger {
	return New(os.Stdout, LevelInfo)
}

// With returns a new logger with the given key-value pairs merged into the context.
func (l *Logger) With(keyVals ...any) *Logger {
	// If the parent logger has a dedicated private buffer (NewForTest),
	// the child inherits it so writes go to the same buffer.
	childBuf := l.buf
	child := &Logger{
		out:   childBuf,
		level: l.level,
		ctx:   make(map[string]any, len(l.ctx)+len(keyVals)/2),
		buf:   childBuf,
	}
	for k, v := range l.ctx {
		child.ctx[k] = v
	}
	for i := 0; i < len(keyVals)-1; i += 2 {
		if k, ok := keyVals[i].(string); ok {
			child.ctx[k] = keyVals[i+1]
		}
	}
	return child
}

// NewForTest creates a logger with a private strings.Builder for test inspection.
func NewForTest(buf *strings.Builder, level Level) *Logger {
	return &Logger{
		out:   buf,
		level: level,
		ctx:   make(map[string]any),
		buf:   buf,
	}
}

// Buffer returns the logger's private buffer, for test assertions.
func (l *Logger) Buffer() *strings.Builder {
	return l.buf
}

func (l *Logger) log(level Level, msg string, keyVals ...any) {
	if level < l.level {
		return
	}
	fields := make(map[string]any, len(l.ctx)+len(keyVals)/2+2)
	for k, v := range l.ctx {
		fields[k] = v
	}
	for i := 0; i < len(keyVals)-1; i += 2 {
		if k, ok := keyVals[i].(string); ok {
			fields[k] = keyVals[i+1]
		}
	}
	fields["msg"] = msg
	fields["level"] = level.String()
	fields["time"] = time.Now().UTC().Format(time.RFC3339)
	l.mu.Lock()
	defer l.mu.Unlock()
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf := l.buf
	buf.Reset()
	for _, k := range keys {
		fmt.Fprintf(buf, "%s=", k)
		if v, ok := fields[k].(string); ok {
			fmt.Fprintf(buf, "%s ", v)
		} else {
			b, _ := json.Marshal(fields[k])
			buf.Write(b)
			buf.WriteByte(' ')
		}
	}
	// Remove trailing space and add newline
	if buf.Len() > 0 {
		b := buf.String()
		buf.Reset()
		buf.WriteString(b[:len(b)-1])
		buf.WriteByte('\n')
	}
	l.out.Write([]byte(buf.String()))
}

// Debug logs at LevelDebug.
func (l *Logger) Debug(msg string, keyVals ...any) { l.log(LevelDebug, msg, keyVals...) }

// Info logs at LevelInfo.
func (l *Logger) Info(msg string, keyVals ...any) { l.log(LevelInfo, msg, keyVals...) }

// Warn logs at LevelWarn.
func (l *Logger) Warn(msg string, keyVals ...any) { l.log(LevelWarn, msg, keyVals...) }

// Error logs at LevelError.
func (l *Logger) Error(msg string, keyVals ...any) { l.log(LevelError, msg, keyVals...) }
