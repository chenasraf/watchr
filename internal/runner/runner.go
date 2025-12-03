package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// Line represents a single line of output with its line number
type Line struct {
	Number  int
	Content string
}

// FormatLine returns the formatted line with line number
func (l Line) FormatLine(width int, showLineNum bool) string {
	if !showLineNum {
		return l.Content
	}
	return fmt.Sprintf("%*d  %s", width, l.Number, l.Content)
}

// Runner executes commands and captures output
type Runner struct {
	Shell   string
	Command string
}

// NewRunner creates a new Runner
func NewRunner(shell, command string) *Runner {
	return &Runner{
		Shell:   shell,
		Command: command,
	}
}

// Run executes the command and returns output lines
func (r *Runner) Run(ctx context.Context) ([]Line, error) {
	cmd := exec.CommandContext(ctx, r.Shell, "-c", r.Command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	var lines []Line
	lineNum := 1

	// Read stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		lines = append(lines, Line{
			Number:  lineNum,
			Content: scanner.Text(),
		})
		lineNum++
	}

	// Read stderr
	stderrScanner := bufio.NewScanner(stderr)
	for stderrScanner.Scan() {
		lines = append(lines, Line{
			Number:  lineNum,
			Content: stderrScanner.Text(),
		})
		lineNum++
	}

	// Wait for command to finish (ignore exit code - we still want to show output)
	_ = cmd.Wait()

	return lines, nil
}

// RunStreaming executes the command and streams output lines to the callback
// The callback is called for each line as it arrives
func (r *Runner) RunStreaming(ctx context.Context, lines *[]Line, mu *sync.RWMutex) error {
	cmd := exec.CommandContext(ctx, r.Shell, "-c", r.Command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	lineNum := 1

	// Read from both stdout and stderr concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	readPipe := func(pipe io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			mu.Lock()
			*lines = append(*lines, Line{
				Number:  lineNum,
				Content: scanner.Text(),
			})
			lineNum++
			mu.Unlock()
		}
	}

	go readPipe(stdout)
	go readPipe(stderr)

	wg.Wait()

	// Wait for command to finish (ignore exit code - we still want to show output)
	_ = cmd.Wait()

	return nil
}

// RunSimple executes the command and returns output as string slice
func (r *Runner) RunSimple(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, r.Shell, "-c", r.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Still return output even on error (non-zero exit)
		if len(output) > 0 {
			return splitLines(string(output)), nil
		}
		return nil, fmt.Errorf("command failed: %w", err)
	}
	return splitLines(string(output)), nil
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
