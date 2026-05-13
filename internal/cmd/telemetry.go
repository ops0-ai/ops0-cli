package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/spf13/cobra"
)

// `ops0 telemetry blocked-command "<cmd>"` is called by the PreToolUse Bash
// hook right before it exits 2. We keep this as a CLI subcommand (rather
// than having the hook curl directly) so credentials stay in
// ~/.ops0/config.yaml and never leak into a shell script.
//
// Design rule: this command MUST exit 0 in all telemetry-related failure
// modes (network down, not logged in, etc.) so a flaky network never
// converts a successful block into an apparent allow. The agent's exit
// 2 still comes from the hook script's separate `exit 2` line.

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Internal: report CLI events to the ops0 audit trail",
	Long: `Subcommands here are invoked by ops0-installed hooks (Claude Code
PostToolUse / PreToolUse) to persist events into the org's audit trail.

You generally don't need to run these by hand.`,
}

var (
	blockedCmdPattern string
	blockedCmdTitle   string
)

var blockedCommandCmd = &cobra.Command{
	Use:   "blocked-command <command>",
	Short: "Record a destructive command that was just blocked by a hook",
	Long: `Posts a single audit-trail row for a command that the PreToolUse hook
blocked. Used by the hook installed by ` + "`ops0 init`" + `.

This is always best-effort — telemetry never affects the exit code. The
hook script handles the actual block via its own ` + "`exit 2`" + `.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runBlockedCommand,
}

func init() {
	blockedCommandCmd.Flags().StringVar(&blockedCmdPattern, "pattern", "", "Pattern that matched (e.g. 'terraform destroy')")
	blockedCommandCmd.Flags().StringVar(&blockedCmdTitle, "title", "Destructive command blocked", "Short title shown in the audit table")
	telemetryCmd.AddCommand(blockedCommandCmd)
}

func runBlockedCommand(_ *cobra.Command, args []string) error {
	// We intentionally swallow all errors below — telemetry is best-effort,
	// the hook will still exit 2 to block the agent. Returning non-zero here
	// would propagate into the hook's overall exit code and could mask the
	// actual block intent.
	cfg, err := config.LoadUser()
	if err != nil || cfg.APIKey == "" {
		return nil
	}

	command := args[0]

	// Hash the cwd so the audit row can attribute "this came from THIS
	// laptop's checkout of repo X" without storing the actual path.
	cwd, _ := os.Getwd()
	hash := sha256.Sum256([]byte(cwd))

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	if err := client.ReportBlockedCommand(&api.BlockedCommand{
		Command:        command,
		MatchedPattern: blockedCmdPattern,
		Title:          blockedCmdTitle,
		RepoHash:       hex.EncodeToString(hash[:]),
		CLIVersion:     buildVersion,
	}); err != nil {
		// Print to stderr so a debug-curious user sees it, but never fail.
		fmt.Fprintf(os.Stderr, "ops0 telemetry: best-effort post failed (%v) — block still in effect\n", err)
	}
	return nil
}
