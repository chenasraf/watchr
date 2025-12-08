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

	// Wait for command to finish and get exit code
	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return Result{Lines: lines, ExitCode: exitCode}, nil
}

// RunStreaming executes the command and streams output lines to the callback
// The callback is called for each line as it arrives
func (r *Runner) RunStreaming(ctx context.Context, lines *[]Line, mu *sync.RWMutex) error {
	args := r.buildCommand()
	cmd := exec.CommandContext(ctx, r.Shell, args...)

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
	args := r.buildCommand()
	cmd := exec.CommandContext(ctx, r.Shell, args...)
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
