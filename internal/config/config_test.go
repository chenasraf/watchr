package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func resetViper() {
	viper.Reset()
}

// isolateConfig sets up a clean environment for config tests by:
// - Resetting viper
// - Changing to a temp directory
// - Setting XDG_CONFIG_HOME to the temp directory
// Returns the temp directory path and a cleanup function.
func isolateConfig(t *testing.T) (string, func()) {
	t.Helper()
	resetViper()

	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	cleanup := func() {
		_ = os.Chdir(oldWd)
		_ = os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}

	return tmpDir, cleanup
}

func TestInit(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	Init()

	// Check defaults are set
	if got := viper.GetString(KeyShell); got != "sh" {
		t.Errorf("expected default shell 'sh', got %q", got)
	}

	if got := viper.GetString(KeyPreviewSize); got != "40%" {
		t.Errorf("expected default preview-size '40%%', got %q", got)
	}

	if got := viper.GetString(KeyPreviewPosition); got != "bottom" {
		t.Errorf("expected default preview-position 'bottom', got %q", got)
	}

	if got := viper.GetBool(KeyLineNumbers); got != true {
		t.Errorf("expected default line-numbers true, got %v", got)
	}

	if got := viper.GetInt(KeyLineWidth); got != 6 {
		t.Errorf("expected default line-width 6, got %d", got)
	}

	if got := viper.GetString(KeyPrompt); got != "watchr> " {
		t.Errorf("expected default prompt 'watchr> ', got %q", got)
	}

	if got := viper.GetInt(KeyRefresh); got != 0 {
		t.Errorf("expected default refresh 0, got %d", got)
	}
}

func TestGetters(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	Init()

	if got := GetString(KeyShell); got != "sh" {
		t.Errorf("GetString: expected 'sh', got %q", got)
	}

	if got := GetInt(KeyLineWidth); got != 6 {
		t.Errorf("GetInt: expected 6, got %d", got)
	}

	if got := GetBool(KeyLineNumbers); got != true {
		t.Errorf("GetBool: expected true, got %v", got)
	}
}

func TestShowLineNumbers(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	Init()

	// Default: line numbers enabled
	if got := ShowLineNumbers(); got != true {
		t.Errorf("expected ShowLineNumbers() true by default, got %v", got)
	}

	// When no-line-numbers is set
	viper.Set("no-line-numbers", true)
	if got := ShowLineNumbers(); got != false {
		t.Errorf("expected ShowLineNumbers() false when no-line-numbers=true, got %v", got)
	}

	// Reset
	viper.Set("no-line-numbers", false)
	if got := ShowLineNumbers(); got != true {
		t.Errorf("expected ShowLineNumbers() true when no-line-numbers=false, got %v", got)
	}
}

func TestBindFlags(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	Init()

	// Create a new flag set
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("shell", "sh", "")
	flags.String("preview-size", "40%", "")
	flags.String("preview-position", "bottom", "")
	flags.Int("line-width", 6, "")
	flags.String("prompt", "watchr> ", "")
	flags.Int("refresh", 0, "")
	flags.Bool("no-line-numbers", false, "")

	// Parse with custom values
	err := flags.Parse([]string{"--shell=bash", "--preview-size=50%", "--line-width=8"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	// Bind flags
	BindFlags(flags)

	// Check that flag values override defaults
	if got := GetString(KeyShell); got != "bash" {
		t.Errorf("expected shell 'bash' from flag, got %q", got)
	}

	if got := GetString(KeyPreviewSize); got != "50%" {
		t.Errorf("expected preview-size '50%%' from flag, got %q", got)
	}

	if got := GetInt(KeyLineWidth); got != 8 {
		t.Errorf("expected line-width 8 from flag, got %d", got)
	}

	// Non-overridden values should still be defaults
	if got := GetString(KeyPreviewPosition); got != "bottom" {
		t.Errorf("expected preview-position 'bottom' (default), got %q", got)
	}
}

func TestConfigFileLoading(t *testing.T) {
	tmpDir, cleanup := isolateConfig(t)
	defer cleanup()

	// Create a config file in the temp directory
	configPath := filepath.Join(tmpDir, "watchr.yaml")
	configContent := `shell: zsh
preview-size: "60%"
preview-position: right
line-numbers: true
line-width: 4
prompt: "test> "
refresh: 5
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Initialize config
	Init()

	// Check that config file values are loaded
	if got := GetString(KeyShell); got != "zsh" {
		t.Errorf("expected shell 'zsh' from config file, got %q", got)
	}

	if got := GetString(KeyPreviewSize); got != "60%" {
		t.Errorf("expected preview-size '60%%' from config file, got %q", got)
	}

	if got := GetString(KeyPreviewPosition); got != "right" {
		t.Errorf("expected preview-position 'right' from config file, got %q", got)
	}

	if got := GetInt(KeyLineWidth); got != 4 {
		t.Errorf("expected line-width 4 from config file, got %d", got)
	}

	if got := GetString(KeyPrompt); got != "test> " {
		t.Errorf("expected prompt 'test> ' from config file, got %q", got)
	}

	if got := GetInt(KeyRefresh); got != 5 {
		t.Errorf("expected refresh 5 from config file, got %d", got)
	}

	// ConfigFileUsed should return the path
	if used := ConfigFileUsed(); used == "" {
		t.Error("expected ConfigFileUsed() to return config file path")
	}
}

func TestConfigFileWithFlags(t *testing.T) {
	tmpDir, cleanup := isolateConfig(t)
	defer cleanup()

	// Create a config file in the temp directory
	configPath := filepath.Join(tmpDir, "watchr.yaml")
	configContent := `shell: zsh
preview-size: "60%"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Initialize config
	Init()

	// Create flags and parse with override
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("shell", "sh", "")
	flags.String("preview-size", "40%", "")
	flags.String("preview-position", "bottom", "")
	flags.Int("line-width", 6, "")
	flags.String("prompt", "watchr> ", "")
	flags.Int("refresh", 0, "")
	flags.Bool("no-line-numbers", false, "")

	// Override shell via flag
	err := flags.Parse([]string{"--shell=bash"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	BindFlags(flags)

	// Flag should override config file
	if got := GetString(KeyShell); got != "bash" {
		t.Errorf("expected shell 'bash' (flag override), got %q", got)
	}

	// Config file value should be used when no flag override
	if got := GetString(KeyPreviewSize); got != "60%" {
		t.Errorf("expected preview-size '60%%' (from config file), got %q", got)
	}

	// Default should be used when not in config file and no flag
	if got := GetString(KeyPreviewPosition); got != "bottom" {
		t.Errorf("expected preview-position 'bottom' (default), got %q", got)
	}
}

func TestGetConfigDir(t *testing.T) {
	dir := getConfigDir()
	if dir == "" {
		t.Log("getConfigDir returned empty string (may be expected in some environments)")
	} else {
		t.Logf("getConfigDir returned: %s", dir)
	}
}

func TestInitWithFile(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	// Create a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom.yaml")
	configContent := `shell: fish
preview-size: "75%"
preview-position: left
line-numbers: false
line-width: 8
prompt: "custom> "
refresh: 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Initialize with specific file
	if err := InitWithFile(configPath); err != nil {
		t.Fatalf("InitWithFile failed: %v", err)
	}

	// Check values from config file
	if got := GetString(KeyShell); got != "fish" {
		t.Errorf("expected shell 'fish', got %q", got)
	}

	if got := GetString(KeyPreviewSize); got != "75%" {
		t.Errorf("expected preview-size '75%%', got %q", got)
	}

	if got := GetString(KeyPreviewPosition); got != "left" {
		t.Errorf("expected preview-position 'left', got %q", got)
	}

	if got := GetBool(KeyLineNumbers); got != false {
		t.Errorf("expected line-numbers false, got %v", got)
	}

	if got := GetInt(KeyLineWidth); got != 8 {
		t.Errorf("expected line-width 8, got %d", got)
	}

	if got := GetString(KeyPrompt); got != "custom> " {
		t.Errorf("expected prompt 'custom> ', got %q", got)
	}

	if got := GetInt(KeyRefresh); got != 10 {
		t.Errorf("expected refresh 10, got %d", got)
	}

	// ConfigFileUsed should return the specified path
	if used := ConfigFileUsed(); used != configPath {
		t.Errorf("expected ConfigFileUsed() = %q, got %q", configPath, used)
	}
}

func TestInitWithFileNotFound(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	err := InitWithFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestInitWithFileTOML(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	// Create a TOML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	configContent := `shell = "zsh"
preview-size = "50%"
preview-position = "top"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := InitWithFile(configPath); err != nil {
		t.Fatalf("InitWithFile failed for TOML: %v", err)
	}

	if got := GetString(KeyShell); got != "zsh" {
		t.Errorf("expected shell 'zsh', got %q", got)
	}

	if got := GetString(KeyPreviewPosition); got != "top" {
		t.Errorf("expected preview-position 'top', got %q", got)
	}
}

func TestInitWithFileJSON(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	// Create a JSON config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configContent := `{
  "shell": "bash",
  "preview-size": "30%",
  "preview-position": "right"
}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := InitWithFile(configPath); err != nil {
		t.Fatalf("InitWithFile failed for JSON: %v", err)
	}

	if got := GetString(KeyShell); got != "bash" {
		t.Errorf("expected shell 'bash', got %q", got)
	}

	if got := GetString(KeyPreviewSize); got != "30%" {
		t.Errorf("expected preview-size '30%%', got %q", got)
	}
}

func TestInitWithFileDefaults(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	// Create a config file with only some values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.yaml")
	configContent := `shell: fish
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := InitWithFile(configPath); err != nil {
		t.Fatalf("InitWithFile failed: %v", err)
	}

	// Specified value should be loaded
	if got := GetString(KeyShell); got != "fish" {
		t.Errorf("expected shell 'fish', got %q", got)
	}

	// Unspecified values should use defaults
	if got := GetString(KeyPreviewSize); got != "40%" {
		t.Errorf("expected default preview-size '40%%', got %q", got)
	}

	if got := GetString(KeyPreviewPosition); got != "bottom" {
		t.Errorf("expected default preview-position 'bottom', got %q", got)
	}

	if got := GetInt(KeyLineWidth); got != 6 {
		t.Errorf("expected default line-width 6, got %d", got)
	}
}
