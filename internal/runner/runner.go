package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// sanitizeLine removes control sequences that can corrupt terminal rendering
func sanitizeLine(s string) string {
	// Remove carriage returns
	s = strings.ReplaceAll(s, "\r", "")
	// Convert tabs to spaces (tabs cause width calculation issues)
	s = strings.ReplaceAll(s, "\t", "        ")
	return s
}

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
	Shell       string
	Command     string
	Interactive bool
}

// NewRunner creates a new Runner
func NewRunner(shell, command string) *Runner {
	return &Runner{
		Shell:       shell,
		Command:     command,
		Interactive: false,
	}
}

// NewInteractiveRunner creates a new Runner that sources shell rc files
func NewInteractiveRunner(shell, command string) *Runner {
	return &Runner{
		Shell:       shell,
		Command:     command,
		Interactive: true,
	}
}

// buildCommand returns the shell arguments for executing the command.
// If Interactive is true, it wraps the command to source the appropriate rc file.
func (r *Runner) buildCommand() []string {
	if !r.Interactive {
		return []string{"-c", r.Command}
	}

	// For interactive mode, source the appropriate rc file before running the command
	rcFile := r.getRCFile()
	if rcFile != "" {
		// Source the rc file if it exists, then run the command
		wrappedCmd := fmt.Sprintf("[ -f %s ] && . %s; %s", rcFile, rcFile, r.Command)
		return []string{"-c", wrappedCmd}
	}

	return []string{"-c", r.Command}
}

// getRCFile returns the path to the shell's rc file based on the shell being used.
func (r *Runner) getRCFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	shellBase := filepath.Base(r.Shell)
	switch shellBase {
	case "bash":
		// Prefer .bashrc for interactive settings, fall back to .bash_profile
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		return filepath.Join(home, ".bash_profile")
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "fish":
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "fish", "config.fish")
	case "ksh":
		return filepath.Join(home, ".kshrc")
	case "sh":
		// POSIX sh uses ENV variable or .profile
		if env := os.Getenv("ENV"); env != "" {
			return env
		}
		return filepath.Join(home, ".profile")
	default:
		// Try common patterns for unknown shells
		return filepath.Join(home, "."+shellBase+"rc")
	}
}

// Result contains the output and exit code of a command run
type Result struct {
	Lines    []Line
	ExitCode int
}

// Run executes the command and returns output lines with exit code
func (r *Runner) Run(ctx context.Context) (Result, error) {
	args := r.buildCommand()
	cmd := exec.CommandContext(ctx, r.Shell, args...)
	cmd.Env = append(os.Environ(), "WATCHR=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("failed to start command: %w", err)
	}

	var lines []Line
	lineNum := 1

	// Read stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		lines = append(lines, Line{
			Number:  lineNum,
			Content: sanitizeLine(scanner.Text()),
		})
		lineNum++
	}

	// Read stderr
	stderrScanner := bufio.NewScanner(stderr)
	for stderrScanner.Scan() {
		lines = append(lines, Line{
			Number:  lineNum,
			Content: sanitizeLine(stderrScanner.Text()),
		})
		lineNum++
	}

	// Wait for command to finish and get exit code
	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return Result{Lines: lines, ExitCode: exitCode}, nil
}

// StreamingResult holds the state of a streaming command
type StreamingResult struct {
	Lines            *[]Line
	ExitCode         int
	Done             bool
	Error            error
	PrevLineCount    int // Number of lines from previous run (for trimming)
	CurrentLineCount int // Number of lines written by current run
	mu               sync.RWMutex
}

// GetLines returns a copy of the current lines (thread-safe)
func (s *StreamingResult) GetLines() []Line {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Lines == nil {
		return nil
	}
	result := make([]Line, len(*s.Lines))
	copy(result, *s.Lines)
	return result
}

// LineCount returns the current number of lines (thread-safe)
func (s *StreamingResult) LineCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Lines == nil {
		return 0
	}
	return len(*s.Lines)
}

// IsDone returns whether the command has finished (thread-safe)
func (s *StreamingResult) IsDone() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Done
}

// GetCurrentLineCount returns the number of lines written by the current run (thread-safe)
func (s *StreamingResult) GetCurrentLineCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentLineCount
}

// RunStreaming executes the command and streams output lines in the background.
// Returns a StreamingResult that can be polled for updates.
// The command runs until ctx is cancelled or it completes naturally.
// If prevLines is provided, lines are updated in place rather than starting fresh.
func (r *Runner) RunStreaming(ctx context.Context, prevLines []Line) *StreamingResult {
	// Copy previous lines to allow in-place updates
	lines := make([]Line, len(prevLines))
	copy(lines, prevLines)

	result := &StreamingResult{
		Lines:         &lines,
		ExitCode:      -1,
		Done:          false,
		PrevLineCount: len(prevLines),
	}

	go func() {
		args := r.buildCommand()
		cmd := exec.CommandContext(ctx, r.Shell, args...)
		cmd.Env = append(os.Environ(), "WATCHR=1")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			result.mu.Lock()
			result.Error = fmt.Errorf("failed to create stdout pipe: %w", err)
			result.Done = true
			result.mu.Unlock()
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			result.mu.Lock()
			result.Error = fmt.Errorf("failed to create stderr pipe: %w", err)
			result.Done = true
			result.mu.Unlock()
			return
		}

		if err := cmd.Start(); err != nil {
			result.mu.Lock()
			result.Error = fmt.Errorf("failed to start command: %w", err)
			result.Done = true
			result.mu.Unlock()
			return
		}

		lineNum := 1
		var lineNumMu sync.Mutex

		// Read from both stdout and stderr concurrently
		var wg sync.WaitGroup
		wg.Add(2)

		readPipe := func(pipe io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(pipe)
			for scanner.Scan() {
				lineNumMu.Lock()
				currentLineNum := lineNum
				lineIdx := lineNum - 1 // 0-indexed
				lineNum++
				lineNumMu.Unlock()

				newLine := Line{
					Number:  currentLineNum,
					Content: sanitizeLine(scanner.Text()),
				}

				result.mu.Lock()
				if lineIdx < len(*result.Lines) {
					// Update existing line in place
					(*result.Lines)[lineIdx] = newLine
				} else {
					// Append new line
					*result.Lines = append(*result.Lines, newLine)
				}
				// Track how many lines this run has produced
				if currentLineNum > result.CurrentLineCount {
					result.CurrentLineCount = currentLineNum
				}
				result.mu.Unlock()
			}
		}

		go readPipe(stdout)
		go readPipe(stderr)

		wg.Wait()

		// Wait for command to finish and get exit code
		exitCode := 0
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else if ctx.Err() != nil {
				// Context was cancelled
				exitCode = -1
			}
		}

		result.mu.Lock()
		result.ExitCode = exitCode
		result.Done = true
		result.mu.Unlock()
	}()

	return result
}

// RunSimple executes the command and returns output as string slice
func (r *Runner) RunSimple(ctx context.Context) ([]string, error) {
	args := r.buildCommand()
	cmd := exec.CommandContext(ctx, r.Shell, args...)
	cmd.Env = append(os.Environ(), "WATCHR=1")
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
