package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/chenasraf/watchr/internal/ui"
	flag "github.com/spf13/pflag"
)

//go:embed version.txt
var version string

func main() {
	var (
		showVersion     bool
		showHelp        bool
		previewHeight   int
		previewPosition string
		noLineNumbers   bool
		lineNumWidth    int
		prompt          string
		shell           string
		refreshSeconds  int
	)

	flag.BoolVarP(&showVersion, "version", "v", false, "Show version")
	flag.BoolVarP(&showHelp, "help", "h", false, "Show help")
	flag.IntVarP(&previewHeight, "preview-height", "P", 40, "Preview window height/width percentage (1-100)")
	flag.StringVar(&previewPosition, "preview-position", "bottom", "Preview position: bottom, top, left, right")
	flag.BoolVarP(&noLineNumbers, "no-line-numbers", "n", false, "Disable line numbers")
	flag.IntVarP(&lineNumWidth, "line-width", "w", 6, "Line number width")
	flag.StringVarP(&prompt, "prompt", "p", "watchr> ", "Prompt string")
	flag.StringVarP(&shell, "shell", "s", "sh", "Shell to use for executing commands")
	flag.IntVarP(&refreshSeconds, "refresh", "r", 0, "Auto-refresh interval in seconds (0 = disabled)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: watchr [options] <command to run>\n\n")
		fmt.Fprintf(os.Stderr, "A terminal UI for running and watching command output.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nKeybindings:\n")
		fmt.Fprintf(os.Stderr, "  r, Ctrl-r      Reload (re-run command)\n")
		fmt.Fprintf(os.Stderr, "  q, Esc         Quit\n")
		fmt.Fprintf(os.Stderr, "  j, k           Move down/up\n")
		fmt.Fprintf(os.Stderr, "  g              Go to first line\n")
		fmt.Fprintf(os.Stderr, "  G              Go to last line\n")
		fmt.Fprintf(os.Stderr, "  Ctrl-d/u       Half page down/up\n")
		fmt.Fprintf(os.Stderr, "  PgDn/Up, ^f/b  Full page down/up\n")
		fmt.Fprintf(os.Stderr, "  p              Toggle preview\n")
		fmt.Fprintf(os.Stderr, "  /              Enter filter mode\n")
		fmt.Fprintf(os.Stderr, "  Esc            Exit filter mode / clear filter\n")
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("watchr %s\n", strings.TrimSpace(version))
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command provided")
		flag.Usage()
		os.Exit(1)
	}

	cmdStr := strings.Join(args, " ")

	config := ui.Config{
		Command:         cmdStr,
		Shell:           shell,
		PreviewHeight:   previewHeight,
		PreviewPosition: ui.PreviewPosition(previewPosition),
		ShowLineNums:    !noLineNumbers,
		LineNumWidth:    lineNumWidth,
		Prompt:          prompt,
		RefreshSeconds:  refreshSeconds,
	}

	if err := ui.Run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
