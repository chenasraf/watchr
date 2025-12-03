package main

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chenasraf/watchr/internal/config"
	"github.com/chenasraf/watchr/internal/ui"
	flag "github.com/spf13/pflag"
)

//go:embed version.txt
var version string

func main() {
	var (
		showVersion bool
		showHelp    bool
		showConfig  bool
		configFile  string
	)

	// Define flags (defaults shown in help, but actual defaults come from config)
	flag.BoolVarP(&showVersion, "version", "v", false, "Show version")
	flag.BoolVarP(&showHelp, "help", "h", false, "Show help")
	flag.BoolVarP(&showConfig, "show-config", "C", false, "Show loaded configuration and exit")
	flag.StringVarP(&configFile, "config", "c", "", "Load config from specified path")
	flag.StringP("preview-size", "P", "40%", "Preview size: number for lines/cols, or number% for percentage (e.g., 10 or 40%)")
	flag.StringP("preview-position", "o", "bottom", "Preview position: bottom, top, left, right")
	flag.BoolP("no-line-numbers", "n", false, "Disable line numbers")
	flag.IntP("line-width", "w", 6, "Line number width")
	flag.StringP("prompt", "p", "watchr> ", "Prompt string")
	flag.StringP("shell", "s", "sh", "Shell to use for executing commands")
	flag.IntP("refresh", "r", 0, "Auto-refresh interval in seconds (0 = disabled)")

	printUsage := func(w *os.File) {
		_, _ = fmt.Fprintf(w, "Usage: watchr [options] <command to run>\n\n")
		_, _ = fmt.Fprintf(w, "A terminal UI for running and watching command output.\n\n")
		_, _ = fmt.Fprintf(w, "Options:\n")
		flag.CommandLine.SetOutput(w)
		flag.PrintDefaults()
		flag.CommandLine.SetOutput(os.Stderr)
		_, _ = fmt.Fprintf(w, "\nKeybindings:\n")
		_, _ = fmt.Fprintf(w, "  r, Ctrl-r      Reload (re-run command)\n")
		_, _ = fmt.Fprintf(w, "  q, Esc         Quit\n")
		_, _ = fmt.Fprintf(w, "  j, k           Move down/up\n")
		_, _ = fmt.Fprintf(w, "  g              Go to first line\n")
		_, _ = fmt.Fprintf(w, "  G              Go to last line\n")
		_, _ = fmt.Fprintf(w, "  Ctrl-d/u       Half page down/up\n")
		_, _ = fmt.Fprintf(w, "  PgDn/Up, ^f/b  Full page down/up\n")
		_, _ = fmt.Fprintf(w, "  p              Toggle preview\n")
		_, _ = fmt.Fprintf(w, "  /              Enter filter mode\n")
		_, _ = fmt.Fprintf(w, "  Esc            Exit filter mode / clear filter\n")
		_, _ = fmt.Fprintf(w, "  y              Yank (copy) selected line\n")
	}

	flag.Usage = func() {
		printUsage(os.Stderr)
	}

	flag.Parse()

	// Initialize config (loads config files and sets defaults)
	if configFile != "" {
		if err := config.InitWithFile(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
			os.Exit(1)
		}
	} else {
		config.Init()
	}

	// Bind flags to config (CLI flags override config file values)
	config.BindFlags(flag.CommandLine)

	if showHelp {
		printUsage(os.Stdout)
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("watchr %s\n", strings.TrimSpace(version))
		os.Exit(0)
	}

	if showConfig {
		config.PrintConfig()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command provided")
		flag.Usage()
		os.Exit(1)
	}

	cmdStr := strings.Join(args, " ")

	// Get config values (merged from: defaults < config file < CLI flags)
	previewSize := config.GetString(config.KeyPreviewSize)
	previewPosition := config.GetString(config.KeyPreviewPosition)
	shell := config.GetString(config.KeyShell)
	lineNumWidth := config.GetInt(config.KeyLineWidth)
	prompt := config.GetString(config.KeyPrompt)
	refreshSeconds := config.GetInt(config.KeyRefresh)
	showLineNums := config.ShowLineNumbers()

	// Parse preview size (e.g., "40" for lines/cols, "40%" for percentage)
	previewSizeIsPercent := strings.HasSuffix(previewSize, "%")
	previewSizeStr := strings.TrimSuffix(previewSize, "%")
	previewSizeVal, err := strconv.Atoi(previewSizeStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid preview size: %s\n", previewSize)
		os.Exit(1)
	}

	uiConfig := ui.Config{
		Command:              cmdStr,
		Shell:                shell,
		PreviewSize:          previewSizeVal,
		PreviewSizeIsPercent: previewSizeIsPercent,
		PreviewPosition:      ui.PreviewPosition(previewPosition),
		ShowLineNums:         showLineNums,
		LineNumWidth:         lineNumWidth,
		Prompt:               prompt,
		RefreshSeconds:       refreshSeconds,
	}

	if err := ui.Run(uiConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
