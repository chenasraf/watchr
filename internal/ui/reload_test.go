package ui

import (
	"testing"
	"time"

	"github.com/chenasraf/watchr/internal/runner"
)

func waitDone(t *testing.T, m *model) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if m.streamResult != nil && m.streamResult.IsDone() {
			// fire one tick after done so model state syncs
			_, _ = m.Update(streamTickMsg(time.Now()))
			return
		}
		_, _ = m.Update(streamTickMsg(time.Now()))
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("stream did not complete")
}

// TestReload_SameLineCountUpdatesInPlace verifies that when the new run
// produces the same number of lines as the previous run, the new content
// (delivered via in-place updates) is reflected in m.lines.
func TestReload_SameLineCountUpdatesInPlace(t *testing.T) {
	cfg := Config{Command: "echo new1; echo new2; echo new3", Shell: "sh"}
	m := testModel(cfg)
	m.height = 30
	m.width = 80

	// Pretend a previous run left 3 lines of stale content
	m.lines = []runner.Line{
		{Number: 1, Content: "old1"},
		{Number: 2, Content: "old2"},
		{Number: 3, Content: "old3"},
	}
	m.lastLineCount = 3
	m.updateFiltered()

	_, _ = m.actionReload()
	waitDone(t, m)

	if len(m.lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(m.lines))
	}
	for i, want := range []string{"new1", "new2", "new3"} {
		if m.lines[i].Content != want {
			t.Errorf("line %d: expected %q, got %q", i, want, m.lines[i].Content)
		}
	}
}

// TestReload_FewerLinesTrimsToNewRun verifies that when the new run produces
// fewer lines than the previous run, m.lines is trimmed AND shows the new
// content in the surviving slots (not stale prev content).
func TestReload_FewerLinesTrimsToNewRun(t *testing.T) {
	cfg := Config{Command: "echo new1; echo new2", Shell: "sh"}
	m := testModel(cfg)
	m.height = 30
	m.width = 80

	m.lines = []runner.Line{
		{Number: 1, Content: "old1"},
		{Number: 2, Content: "old2"},
		{Number: 3, Content: "old3"},
		{Number: 4, Content: "old4"},
	}
	m.lastLineCount = 4
	m.updateFiltered()

	_, _ = m.actionReload()
	waitDone(t, m)

	if len(m.lines) != 2 {
		t.Fatalf("expected 2 lines after reload, got %d", len(m.lines))
	}
	for i, want := range []string{"new1", "new2"} {
		if m.lines[i].Content != want {
			t.Errorf("line %d: expected %q, got %q", i, want, m.lines[i].Content)
		}
	}
}

// TestReload_MoreLines verifies that when the new run produces more lines
// than the previous run, all new lines are visible.
func TestReload_MoreLines(t *testing.T) {
	cfg := Config{Command: "echo a; echo b; echo c; echo d; echo e", Shell: "sh"}
	m := testModel(cfg)
	m.height = 30
	m.width = 80

	m.lines = []runner.Line{
		{Number: 1, Content: "old1"},
		{Number: 2, Content: "old2"},
		{Number: 3, Content: "old3"},
	}
	m.lastLineCount = 3
	m.updateFiltered()

	_, _ = m.actionReload()
	waitDone(t, m)

	if len(m.lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(m.lines))
	}
	for i, want := range []string{"a", "b", "c", "d", "e"} {
		if m.lines[i].Content != want {
			t.Errorf("line %d: expected %q, got %q", i, want, m.lines[i].Content)
		}
	}
}
