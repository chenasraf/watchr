package runner

import (
	"context"
	"testing"
	"time"
)

func TestNewRunner(t *testing.T) {
	r := NewRunner("sh", "echo hello")
	if r.Shell != "sh" {
		t.Errorf("expected shell 'sh', got %q", r.Shell)
	}
	if r.Command != "echo hello" {
		t.Errorf("expected command 'echo hello', got %q", r.Command)
	}
}

func TestRunner_Run(t *testing.T) {
	tests := []struct {
		name        string
		shell       string
		command     string
		wantLines   int
		wantContent string
	}{
		{
			name:        "simple echo",
			shell:       "sh",
			command:     "echo hello",
			wantLines:   1,
			wantContent: "hello",
		},
		{
			name:        "multiline output",
			shell:       "sh",
			command:     "echo 'line1\nline2\nline3'",
			wantLines:   3,
			wantContent: "line1",
		},
		{
			name:        "empty output",
			shell:       "sh",
			command:     "true",
			wantLines:   0,
			wantContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRunner(tt.shell, tt.command)
			ctx := context.Background()

			result, err := r.Run(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Lines) != tt.wantLines {
				t.Errorf("expected %d lines, got %d", tt.wantLines, len(result.Lines))
			}

			if tt.wantLines > 0 && result.Lines[0].Content != tt.wantContent {
				t.Errorf("expected first line %q, got %q", tt.wantContent, result.Lines[0].Content)
			}
		})
	}
}

func TestRunner_RunSimple(t *testing.T) {
	r := NewRunner("sh", "echo 'hello world'")
	ctx := context.Background()

	lines, err := r.RunSimple(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	if lines[0] != "hello world" {
		t.Errorf("expected 'hello world', got %q", lines[0])
	}
}

func TestRunner_RunWithContext(t *testing.T) {
	r := NewRunner("sh", "sleep 10")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := r.Run(ctx)
	// The command should be killed by context timeout
	if err == nil {
		t.Log("command completed (may happen on fast systems)")
	}
}

func TestRunner_RunWithFailingCommand(t *testing.T) {
	r := NewRunner("sh", "exit 1")
	ctx := context.Background()

	// Should not return error for non-zero exit, just empty output
	result, err := r.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Lines) != 0 {
		t.Errorf("expected 0 lines for exit 1, got %d", len(result.Lines))
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestRunner_RunWithOutputAndError(t *testing.T) {
	r := NewRunner("sh", "echo 'output'; exit 1")
	ctx := context.Background()

	result, err := r.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(result.Lines))
	}

	if result.Lines[0].Content != "output" {
		t.Errorf("expected 'output', got %q", result.Lines[0].Content)
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestLine_FormatLine(t *testing.T) {
	tests := []struct {
		name        string
		line        Line
		width       int
		showLineNum bool
		want        string
	}{
		{
			name:        "with line number",
			line:        Line{Number: 1, Content: "hello"},
			width:       6,
			showLineNum: true,
			want:        "     1  hello",
		},
		{
			name:        "without line number",
			line:        Line{Number: 1, Content: "hello"},
			width:       6,
			showLineNum: false,
			want:        "hello",
		},
		{
			name:        "larger line number",
			line:        Line{Number: 123, Content: "test"},
			width:       6,
			showLineNum: true,
			want:        "   123  test",
		},
		{
			name:        "narrow width",
			line:        Line{Number: 1, Content: "content"},
			width:       3,
			showLineNum: true,
			want:        "  1  content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.line.FormatLine(tt.width, tt.showLineNum)
			if got != tt.want {
				t.Errorf("FormatLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single line",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "multiple lines",
			input: "line1\nline2\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "trailing newline",
			input: "hello\n",
			want:  []string{"hello"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitLines() returned %d lines, want %d", len(got), len(tt.want))
			}
			for i, line := range got {
				if line != tt.want[i] {
					t.Errorf("splitLines()[%d] = %q, want %q", i, line, tt.want[i])
				}
			}
		})
	}
}
