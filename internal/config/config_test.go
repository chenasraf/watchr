package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	if got := viper.GetString(KeyRefresh); got != "0" {
		t.Errorf("expected default refresh '0', got %q", got)
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
	flags.String("refresh", "0", "")
	flags.Bool("refresh-from-start", false, "")
	flags.Bool("no-line-numbers", false, "")
	flags.Bool("interactive", false, "")

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

	if got := GetDuration(KeyRefresh); got != 5*time.Second {
		t.Errorf("expected refresh 5s from config file, got %v", got)
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
	flags.String("refresh", "0", "")
	flags.Bool("refresh-from-start", false, "")
	flags.Bool("no-line-numbers", false, "")
	flags.Bool("interactive", false, "")

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

	if got := GetDuration(KeyRefresh); got != 10*time.Second {
		t.Errorf("expected refresh 10s, got %v", got)
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

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		// Empty and zero values
		{"", 0, false},
		{"0", 0, false},

		// Plain numbers (default to seconds)
		{"1", 1 * time.Second, false},
		{"5", 5 * time.Second, false},
		{"60", 60 * time.Second, false},

		// Decimal seconds
		{"0.5", 500 * time.Millisecond, false},
		{"1.5", 1500 * time.Millisecond, false},
		{"2.25", 2250 * time.Millisecond, false},
		{"0.1", 100 * time.Millisecond, false},
		{"0.001", 1 * time.Millisecond, false},

		// Explicit seconds suffix
		{"1s", 1 * time.Second, false},
		{"5s", 5 * time.Second, false},
		{"0.5s", 500 * time.Millisecond, false},
		{"1.5s", 1500 * time.Millisecond, false},

		// Milliseconds suffix
		{"100ms", 100 * time.Millisecond, false},
		{"500ms", 500 * time.Millisecond, false},
		{"1000ms", 1000 * time.Millisecond, false},
		{"1500ms", 1500 * time.Millisecond, false},
		{"50.5ms", 50500 * time.Microsecond, false},

		// Minutes suffix
		{"1m", 1 * time.Minute, false},
		{"5m", 5 * time.Minute, false},
		{"0.5m", 30 * time.Second, false},
		{"1.5m", 90 * time.Second, false},

		// Hours suffix
		{"1h", 1 * time.Hour, false},
		{"2h", 2 * time.Hour, false},
		{"0.5h", 30 * time.Minute, false},
		{"1.5h", 90 * time.Minute, false},

		// Invalid formats
		{"abc", 0, true},
		{"1d", 0, true},  // days not supported
		{"1w", 0, true},  // weeks not supported
		{"-1", 0, true},  // negative not supported
		{"-1s", 0, true}, // negative not supported
		{"1.2.3", 0, true},
		{"s", 0, true},
		{"ms", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	Init()

	// Test default value
	if got := GetDuration(KeyRefresh); got != 0 {
		t.Errorf("expected default refresh 0, got %v", got)
	}

	// Test with seconds value
	viper.Set(KeyRefresh, "5")
	if got := GetDuration(KeyRefresh); got != 5*time.Second {
		t.Errorf("expected refresh 5s, got %v", got)
	}

	// Test with decimal value
	viper.Set(KeyRefresh, "0.5")
	if got := GetDuration(KeyRefresh); got != 500*time.Millisecond {
		t.Errorf("expected refresh 500ms, got %v", got)
	}

	// Test with explicit seconds suffix
	viper.Set(KeyRefresh, "2s")
	if got := GetDuration(KeyRefresh); got != 2*time.Second {
		t.Errorf("expected refresh 2s, got %v", got)
	}

	// Test with milliseconds
	viper.Set(KeyRefresh, "250ms")
	if got := GetDuration(KeyRefresh); got != 250*time.Millisecond {
		t.Errorf("expected refresh 250ms, got %v", got)
	}

	// Test with invalid value (should return 0)
	viper.Set(KeyRefresh, "invalid")
	if got := GetDuration(KeyRefresh); got != 0 {
		t.Errorf("expected refresh 0 for invalid value, got %v", got)
	}
}

func TestRefreshFromStartDefault(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	Init()

	// Default should be false
	if got := GetBool(KeyRefreshFromStart); got != false {
		t.Errorf("expected default refresh-from-start false, got %v", got)
	}
}

func TestRefreshFromStartFromConfigFile(t *testing.T) {
	tmpDir, cleanup := isolateConfig(t)
	defer cleanup()

	// Create config file with refresh-from-start: true
	configPath := filepath.Join(tmpDir, "watchr.yaml")
	configContent := `refresh-from-start: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	Init()

	if got := GetBool(KeyRefreshFromStart); got != true {
		t.Errorf("expected refresh-from-start true from config file, got %v", got)
	}
}

func TestRefreshFromStartFromFlag(t *testing.T) {
	_, cleanup := isolateConfig(t)
	defer cleanup()

	Init()

	// Create flags and parse
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("shell", "sh", "")
	flags.String("preview-size", "40%", "")
	flags.String("preview-position", "bottom", "")
	flags.Int("line-width", 6, "")
	flags.String("prompt", "watchr> ", "")
	flags.String("refresh", "0", "")
	flags.Bool("refresh-from-start", false, "")
	flags.Bool("no-line-numbers", false, "")
	flags.Bool("interactive", false, "")

	// Parse with refresh-from-start=true
	err := flags.Parse([]string{"--refresh-from-start=true"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	BindFlags(flags)

	if got := GetBool(KeyRefreshFromStart); got != true {
		t.Errorf("expected refresh-from-start true from flag, got %v", got)
	}
}

func TestRefreshFromStartFlagOverridesConfig(t *testing.T) {
	tmpDir, cleanup := isolateConfig(t)
	defer cleanup()

	// Create config file with refresh-from-start: true
	configPath := filepath.Join(tmpDir, "watchr.yaml")
	configContent := `refresh-from-start: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	Init()

	// Create flags and parse with refresh-from-start=false
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("shell", "sh", "")
	flags.String("preview-size", "40%", "")
	flags.String("preview-position", "bottom", "")
	flags.Int("line-width", 6, "")
	flags.String("prompt", "watchr> ", "")
	flags.String("refresh", "0", "")
	flags.Bool("refresh-from-start", false, "")
	flags.Bool("no-line-numbers", false, "")
	flags.Bool("interactive", false, "")

	err := flags.Parse([]string{"--refresh-from-start=false"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	BindFlags(flags)

	// Flag should override config
	if got := GetBool(KeyRefreshFromStart); got != false {
		t.Errorf("expected refresh-from-start false (flag override), got %v", got)
	}
}

func TestRefreshDurationFromConfigFile(t *testing.T) {
	tmpDir, cleanup := isolateConfig(t)
	defer cleanup()

	tests := []struct {
		name     string
		yaml     string
		expected time.Duration
	}{
		{
			name:     "integer seconds",
			yaml:     "refresh: 5\n",
			expected: 5 * time.Second,
		},
		{
			name:     "decimal seconds",
			yaml:     "refresh: 0.5\n",
			expected: 500 * time.Millisecond,
		},
		{
			name:     "string with s suffix",
			yaml:     "refresh: \"2s\"\n",
			expected: 2 * time.Second,
		},
		{
			name:     "string with ms suffix",
			yaml:     "refresh: \"500ms\"\n",
			expected: 500 * time.Millisecond,
		},
		{
			name:     "string decimal with s suffix",
			yaml:     "refresh: \"1.5s\"\n",
			expected: 1500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			configPath := filepath.Join(tmpDir, "watchr.yaml")
			if err := os.WriteFile(configPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			Init()

			got := GetDuration(KeyRefresh)
			if got != tt.expected {
				t.Errorf("GetDuration(KeyRefresh) = %v, want %v", got, tt.expected)
			}
		})
	}
}
