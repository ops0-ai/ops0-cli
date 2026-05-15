package cmd

import (
	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetBuildInfo is called from main() to inject goreleaser-provided build
// metadata. We keep it package-level rather than importing main into version.go.
func SetBuildInfo(v, c, d string) {
	buildVersion = v
	buildCommit = c
	buildDate = d
}

// Execute runs the root command. Called from main.
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "ops0",
	Short: "Governance layer for AI coding assistants",
	Long: `ops0 connects your local AI coding assistants (Claude Code, Codex,
Gemini CLI, etc.) to your organization's policies on the ops0 platform.

Workflow:
  1. ops0 login                Authenticate with an API key from your ops0 settings
  2. ops0 init                 Wire up this repository (writes .ops0/ + CLAUDE.md)
  3. ops0 policies list        Show policies that apply to this repo
  4. ops0 policies check .     Run policies locally against IaC in this directory
  5. ops0 mcp serve            Run the MCP server (point your agent here)

All Rego evaluation happens locally via OPA. Your code never leaves the machine
— only check results (pass/fail counts, anonymized template IDs) are reported
back to ops0 for audit telemetry, and only when you opt in.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Sub-commands register themselves via init() functions in their own files.
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(policiesCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(telemetryCmd)
	rootCmd.AddCommand(versionCmd)
}
