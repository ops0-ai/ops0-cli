// Package main is the entrypoint for the ops0 CLI.
//
// ops0 is the governance layer for AI coding assistants — it lets Claude Code,
// Codex, Gemini CLI, and other agents enforce your organization's policies
// while they generate Infrastructure-as-Code locally.
//
// The CLI is intentionally small: a thin shell over an API client, a local
// Rego evaluator (OPA), and an MCP server. All heavy lifting (policy storage,
// finding correlation, audit telemetry) lives on the ops0 platform.
package main

import (
	"fmt"
	"os"

	"github.com/ops0-ai/ops0-cli/internal/cmd"
)

// Set at build time by goreleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetBuildInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
