package ui

import (
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Command:              "echo test",
		Shell:                "sh",
		PreviewSize:          40,
		PreviewSizeIsPercent: true,
		PreviewPosition:      PreviewBottom,
		ShowLineNums:         true,
		LineNumWidth:         6,
		Prompt:               "watchr> ",
		RefreshInterval:      5 * time.Second,
	}

	if cfg.Command != "echo test" {
		t.Errorf("expected command 'echo test', got %q", cfg.Command)
	}

	if cfg.Shell != "sh" {
		t.Errorf("expected shell 'sh', got %q", cfg.Shell)
	}

	if cfg.PreviewSize != 40 {
		t.Errorf("expected preview size 40, got %d", cfg.PreviewSize)
	}

	if !cfg.PreviewSizeIsPercent {
		t.Error("expected PreviewSizeIsPercent to be true")
	}

	if cfg.PreviewPosition != PreviewBottom {
		t.Errorf("expected preview position 'bottom', got %q", cfg.PreviewPosition)
	}

	if !cfg.ShowLineNums {
		t.Error("expected ShowLineNums to be true")
	}

	if cfg.LineNumWidth != 6 {
		t.Errorf("expected line num width 6, got %d", cfg.LineNumWidth)
	}

	if cfg.Prompt != "watchr> " {
		t.Errorf("expected prompt 'watchr> ', got %q", cfg.Prompt)
	}

	if cfg.RefreshInterval != 5*time.Second {
		t.Errorf("expected refresh interval 5s, got %v", cfg.RefreshInterval)
	}
}

func TestPreviewPositionConstants(t *testing.T) {
	tests := []struct {
		pos  PreviewPosition
		want string
	}{
		{PreviewBottom, "bottom"},
		{PreviewTop, "top"},
		{PreviewLeft, "left"},
		{PreviewRight, "right"},
	}

	for _, tt := range tests {
		if string(tt.pos) != tt.want {
			t.Errorf("PreviewPosition %v != %q", tt.pos, tt.want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test with zero values
	cfg := Config{}

	if cfg.Command != "" {
		t.Errorf("expected empty command, got %q", cfg.Command)
	}

	if cfg.Shell != "" {
		t.Errorf("expected empty shell, got %q", cfg.Shell)
	}

	if cfg.PreviewSize != 0 {
		t.Errorf("expected preview size 0, got %d", cfg.PreviewSize)
	}

	if cfg.PreviewSizeIsPercent {
		t.Error("expected PreviewSizeIsPercent to be false")
	}

	if cfg.PreviewPosition != "" {
		t.Errorf("expected empty preview position, got %q", cfg.PreviewPosition)
	}

	if cfg.ShowLineNums {
		t.Error("expected ShowLineNums to be false")
	}

	if cfg.LineNumWidth != 0 {
		t.Errorf("expected line num width 0, got %d", cfg.LineNumWidth)
	}

	if cfg.Prompt != "" {
		t.Errorf("expected empty prompt, got %q", cfg.Prompt)
	}
}

func TestInitialModel(t *testing.T) {
	cfg := Config{
		Command:              "echo test",
		Shell:                "sh",
		PreviewSize:          40,
		PreviewSizeIsPercent: true,
		PreviewPosition:      PreviewBottom,
		ShowLineNums:         true,
		LineNumWidth:         6,
		Prompt:               "watchr> ",
	}

	m := initialModel(cfg)

	if m.config.Command != cfg.Command {
		t.Errorf("expected command %q, got %q", cfg.Command, m.config.Command)
	}

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}

	if m.offset != 0 {
		t.Errorf("expected offset at 0, got %d", m.offset)
	}

	if m.filterMode {
		t.Error("expected filterMode to be false")
	}

	if m.showPreview {
		t.Error("expected showPreview to be false")
	}

	if !m.loading {
		t.Error("expected loading to be true initially")
	}
}

func TestConfigRefreshFromStart(t *testing.T) {
	// Test with RefreshFromStart false (default)
	cfg := Config{
		Command:          "echo test",
		Shell:            "sh",
		RefreshInterval:  5 * time.Second,
		RefreshFromStart: false,
	}

	if cfg.RefreshFromStart {
		t.Error("expected RefreshFromStart to be false by default")
	}

	// Test with RefreshFromStart true
	cfg.RefreshFromStart = true
	if !cfg.RefreshFromStart {
		t.Error("expected RefreshFromStart to be true after setting")
	}
}

func TestModelUserScrolled(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)

	// Initially should be false
	if m.userScrolled {
		t.Error("expected userScrolled to be false initially")
	}

	// After setting, should be true
	m.userScrolled = true
	if !m.userScrolled {
		t.Error("expected userScrolled to be true after setting")
	}
}

func TestModelRefreshGeneration(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)

	// Initially should be 0
	if m.refreshGeneration != 0 {
		t.Errorf("expected refreshGeneration to be 0 initially, got %d", m.refreshGeneration)
	}

	// After incrementing
	m.refreshGeneration++
	if m.refreshGeneration != 1 {
		t.Errorf("expected refreshGeneration to be 1 after increment, got %d", m.refreshGeneration)
	}
}
