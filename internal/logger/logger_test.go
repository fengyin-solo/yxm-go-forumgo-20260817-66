package logger

import (
	"strings"
	"testing"
)

func TestLoggerDefault(t *testing.T) {
	log := Default()
	if log == nil {
		t.Fatal("Default() returned nil")
	}
	if log.level != LevelInfo {
		t.Errorf("level = %v", log.level)
	}
}

func TestLoggerInfo(t *testing.T) {
	var buf strings.Builder
	log := New(&buf, LevelInfo)
	log.Info("hello world", "key", "value")
	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("output = %q", out)
	}
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("output = %q", out)
	}
}

func TestLoggerDebugNotShown(t *testing.T) {
	var buf strings.Builder
	log := New(&buf, LevelInfo)
	log.Debug("secret debug")
	if buf.Len() > 0 {
		t.Errorf("debug should not be shown at info level: %q", buf.String())
	}
}

func TestLoggerWith(t *testing.T) {
	var buf1 strings.Builder
	log := NewForTest(&buf1, LevelInfo).With("prefix", "test")
	log.Info("message")
	out := buf1.String()
	if !strings.Contains(out, "prefix=test") {
		t.Errorf("output = %q", out)
	}
}

func TestLoggerWithDoesNotMutateParent(t *testing.T) {
	// Parent logger with its own output
	var parentOut strings.Builder
	parent := New(&parentOut, LevelInfo)

	// Child logger with its own buffer (via With)
	child := parent.With("key", "child_value")
	child.Info("from child")

	// Child's buffer should contain its message
	childOut := child.Buffer().String()
	if !strings.Contains(childOut, "child_value") {
		t.Errorf("child output = %q", childOut)
	}
	// Parent's output should NOT contain child's message
	if strings.Contains(parentOut.String(), "child_value") {
		t.Error("parent output should not contain child message")
	}
}

func TestNewForTest(t *testing.T) {
	var buf strings.Builder
	log := NewForTest(&buf, LevelInfo)
	log.Info("test message", "k", "v")
	if buf.Len() == 0 {
		t.Error("NewForTest should write to buffer")
	}
	if log.Buffer() != &buf {
		t.Error("Buffer() should return the private buffer")
	}
}

func TestLoggerCloneIndependence(t *testing.T) {
	var buf strings.Builder
	log := New(&buf, LevelInfo).With("base", "value")
	child := log.With("child", "extra")
	child.Info("child msg")
	childOut := child.Buffer().String()
	if !strings.Contains(childOut, "child=extra") {
		t.Errorf("child output = %q", childOut)
	}
}

func TestLoggerLevelConstants(t *testing.T) {
	if LevelDebug.String() != "DEBUG" {
		t.Errorf("LevelDebug.String() = %q", LevelDebug.String())
	}
	if LevelInfo.String() != "INFO" {
		t.Errorf("LevelInfo.String() = %q", LevelInfo.String())
	}
	if LevelWarn.String() != "WARN" {
		t.Errorf("LevelWarn.String() = %q", LevelWarn.String())
	}
	if LevelError.String() != "ERROR" {
		t.Errorf("LevelError.String() = %q", LevelError.String())
	}
}
