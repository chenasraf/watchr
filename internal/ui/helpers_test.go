package ui

import (
	"context"

	"github.com/chenasraf/watchr/internal/runner"
)

func testModel(cfg Config) *model {
	m := initialModel(cfg)
	return &m
}

func testModelWithLines() *model {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.lines = []runner.Line{
		{Number: 1, Content: "hello world"},
		{Number: 2, Content: "foo bar"},
		{Number: 3, Content: "hello foo"},
		{Number: 4, Content: "baz qux"},
	}
	m.height = 30
	m.width = 80
	m.updateFiltered()
	return m
}

func testModelWithCancel() *model {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := initialModel(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel
	return &m
}
