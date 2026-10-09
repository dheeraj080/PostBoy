package tui

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
)

const goldenSizeW, goldenSizeH = 120, 32

// corner is the first byte of the outer frame's rounded corner (╭).
var corner = []byte("╭")

// TestGoldenDemoView runs the demo model through a real bubbletea program at
// 120x32 and compares the painted frame against testdata/TestGoldenDemoView.golden.
//
// Regenerate the golden file with:
//
//	go test ./internal/tui -run TestGoldenDemoView -update
func TestGoldenDemoView(t *testing.T) {
	// Deterministic colours: the program output goes to a buffer, so the
	// profile must be forced rather than detected.
	oldProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(oldProfile) })

	tm := teatest.NewTestModel(t, New(Options{Demo: true}),
		teatest.WithInitialTermSize(goldenSizeW, goldenSizeH))

	var acc bytes.Buffer
	buf := make([]byte, 8192)
	// Reads from the program output consume it, so everything we care
	// about is accumulated here as it arrives.
	waitFor := func(cond func() bool, what string) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for !cond() {
			n, err := tm.Output().Read(buf)
			if n > 0 {
				acc.Write(buf[:n])
			}
			if err != nil && err != io.EOF {
				t.Fatalf("reading program output: %v", err)
			}
			if time.Now().After(deadline) {
				t.Fatalf("timeout waiting for %s; output:\n%s", what, acc.String())
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	waitFor(func() bool { return bytes.Contains(acc.Bytes(), []byte("201 Created")) },
		"the demo frame (status 201 Created)")
	// The marker sits near the top of the frame; wait until the whole
	// 32-line frame has been written before quitting.
	frameStart := func() int {
		i := bytes.Index(acc.Bytes(), corner)
		if i < 0 {
			return -1
		}
		if j := bytes.LastIndexByte(acc.Bytes()[:i], '\n'); j >= 0 {
			return j + 1
		}
		return i
	}
	waitFor(func() bool {
		s := frameStart()
		return s >= 0 && bytes.Count(acc.Bytes()[s:], []byte("\r\n")) >= goldenSizeH-1
	}, "a complete 32-line frame")

	if err := tm.Quit(); err != nil {
		t.Fatalf("quit: %v", err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	frame := extractGoldenFrame(t, acc.Bytes(), goldenSizeH)
	teatest.RequireEqualOutput(t, frame)
}

// extractGoldenFrame slices the first h-line frame out of a raw program
// output stream: everything before the frame's leading border corner (and the
// escape sequence that starts that line) is dropped, the frame's \r\n line
// separators are normalised to \n, and only the first h lines are kept.
func extractGoldenFrame(t *testing.T, out []byte, h int) []byte {
	t.Helper()
	i := bytes.Index(out, corner)
	if i < 0 {
		t.Fatalf("no frame found in output:\n%s", out)
	}
	// Keep the whole line the corner sits on (its leading SGR sequence).
	if j := bytes.LastIndexByte(out[:i], '\n'); j >= 0 {
		i = j + 1
	}
	lines := bytes.Split(out[i:], []byte("\r\n"))
	if len(lines) < h {
		t.Fatalf("got %d lines, want at least %d", len(lines), h)
	}
	frame := bytes.Join(lines[:h], []byte("\n"))
	if !bytes.Contains(frame, []byte("Ctrl+C")) {
		t.Fatalf("frame does not look complete (no footer):\n%s", frame)
	}
	return frame
}
