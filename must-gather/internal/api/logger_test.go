package api

import (
	"bytes"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestLoggerLogFormatsMessage(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(buf)

	l.Log("hello %s", "world")

	out := buf.String()
	assert.Assert(t, strings.HasSuffix(out, "hello world\n"), "unexpected output: %q", out)
	// A timestamp prefix should be present (date + time separated by a space).
	assert.Assert(t, strings.Count(out, " ") >= 2, "expected a timestamp prefix: %q", out)
}

func TestLoggerWarnAndInfoPrefixes(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(buf)

	l.Warn("something %d", 1)
	l.Info("other %d", 2)

	out := buf.String()
	assert.Assert(t, strings.Contains(out, "WARN: something 1"), "output: %q", out)
	assert.Assert(t, strings.Contains(out, "INFO: other 2"), "output: %q", out)
}

func TestLoggerBeginEnd(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(buf)

	end := l.Begin("doing %s", "work")
	end()

	out := buf.String()
	assert.Assert(t, strings.Contains(out, "BEGIN doing work"), "output: %q", out)
	assert.Assert(t, strings.Contains(out, "END doing work"), "output: %q", out)
	// BEGIN must appear before END.
	assert.Assert(t, strings.Index(out, "BEGIN") < strings.Index(out, "END"), "ordering wrong: %q", out)
}
