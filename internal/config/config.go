package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config keys
const (
	KeyShell           = "shell"
	KeyPreviewSize     = "preview-size"
	KeyPreviewPosition = "preview-position"
	KeyLineNumbers     = "line-numbers"
	KeyLineWidth       = "line-width"
	KeyPrompt          = "prompt"
	KeyRefresh         = "refresh"
)

// Init initializes Viper with config file paths and defaults.
// Should be called before flag.Parse().
func Init() {
	// Set defaults
	viper.SetDefault(KeyShell, "sh")
	viper.SetDefault(KeyPreviewSize, "40%")
	viper.SetDefault(KeyPreviewPosition, "bottom")
	viper.SetDefault(KeyLineNumbers, true)
	viper.SetDefault(KeyLineWidth, 6)
	viper.SetDefault(KeyPrompt, "watchr> ")
	viper.SetDefault(KeyRefresh, 0)

	// Config file name (without extension)
	viper.SetConfigName("watchr")

	// Add config paths in reverse priority order (last added = highest priority)
	// 1. XDG config dir (lowest priority for files)
	if configDir := getConfigDir(); configDir != "" {
		watchrConfigDir := filepath.Join(configDir, "watchr")
		viper.AddConfigPath(watchrConfigDir)
	}

	// 2. Current directory (highest priority for files)
	viper.AddConfigPath(".")

	// Try to read config file (errors are ignored if file doesn't exist)
	_ = viper.ReadInConfig()
}

// BindFlags binds pflags to Viper. Should be called after flag definitions
// but before accessing config values.
func BindFlags(flags *pflag.FlagSet) {
	// Bind each flag to its viper key
	_ = viper.BindPFlag(KeyShell, flags.Lookup("shell"))
	_ = viper.BindPFlag(KeyPreviewSize, flags.Lookup("preview-size"))
	_ = viper.BindPFlag(KeyPreviewPosition, flags.Lookup("preview-position"))
	_ = viper.BindPFlag(KeyLineWidth, flags.Lookup("line-width"))
	_ = viper.BindPFlag(KeyPrompt, flags.Lookup("prompt"))
	_ = viper.BindPFlag(KeyRefresh, flags.Lookup("refresh"))

	// line-numbers is inverted (no-line-numbers flag)
	_ = viper.BindPFlag("no-line-numbers", flags.Lookup("no-line-numbers"))
}

// GetString returns a string config value.
func GetString(key string) string {
	return viper.GetString(key)
}

// GetInt returns an int config value.
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool returns a bool config value.
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// ShowLineNumbers returns whether line numbers should be shown.
// This handles the inverted no-line-numbers flag.
func ShowLineNumbers() bool {
	// If no-line-numbers flag is set, don't show line numbers
	if viper.GetBool("no-line-numbers") {
		return false
	}
	// Otherwise use the line-numbers config value
	return viper.GetBool(KeyLineNumbers)
}

// ConfigFileUsed returns the config file path if one was loaded.
func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

// getConfigDir returns the appropriate config directory for the OS.
func getConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return os.Getenv("APPDATA")
	default:
		// Use XDG_CONFIG_HOME if set, otherwise ~/.config
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return xdg
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".config")
		}
		return ""
	}
}
