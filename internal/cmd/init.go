package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	initProjectID string
	initForce     bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Wire up this repository: writes .ops0/config.json + CLAUDE.md",
	Long: `Initializes the current repository for ops0 governance:

  .ops0/config.json   — Per-repo binding (project ID, paths, policy version).
                        Check this into git so collaborators inherit the same
                        policy set.
  CLAUDE.md           — Appends a section telling Claude Code (and other
                        agents that read CLAUDE.md) to call ops0 before
                        producing IaC. Safe to re-run; the section is fenced
                        so it can be regenerated without clobbering your
                        existing notes.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initProjectID, "project", "", "ops0 IaC project ID to bind this repo to")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite an existing .ops0/config.json")
}

func runInit(cmd *cobra.Command, _ []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	existing, err := config.LoadRepo(cwd)
	if err != nil {
		return err
	}
	if existing != nil && !initForce {
		return fmt.Errorf("already initialized at %s (use --force to overwrite)", config.RepoConfigPath(cwd))
	}

	repoCfg := &config.RepoConfig{
		ProjectID: initProjectID,
	}
	if err := config.SaveRepo(cwd, repoCfg); err != nil {
		return fmt.Errorf("write repo config: %w", err)
	}

	if err := upsertClaudeMd(cwd); err != nil {
		// Don't fail the whole init if CLAUDE.md can't be written —
		// the repo binding is the important part.
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update CLAUDE.md: %v\n", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Initialized ops0 at %s\n", config.RepoConfigPath(cwd))
	if initProjectID == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "  Note: no project ID bound — only org-wide policies will apply.")
		fmt.Fprintln(cmd.OutOrStdout(), "  Re-run with --project=<id> to bind to a specific IaC project.")
	}
	return nil
}

// upsertClaudeMd ensures the repo's CLAUDE.md contains an ops0 governance
// block. The block is fenced with marker comments so we can replace it
// idempotently on subsequent inits.
func upsertClaudeMd(repoRoot string) error {
	path := filepath.Join(repoRoot, "CLAUDE.md")
	const (
		begin = "<!-- ops0:begin -->"
		end   = "<!-- ops0:end -->"
	)
	block := begin + `
## ops0 Governance

This repository is governed by [ops0](https://brew.ops0.ai) policies.

Before generating or modifying Infrastructure-as-Code (Terraform / OpenTofu /
Oxid) in this repository:

1. Call the ` + "`list_policies`" + ` tool on the ops0 MCP server (if available),
   or run ` + "`ops0 policies list`" + ` in a shell, to see the rules that apply.
2. After writing any ` + "`.tf` / `.tofu` / `.hcl`" + ` file, run
   ` + "`ops0 policies check <path>`" + ` and fix any violations before
   considering the task complete.
3. Treat policy violations the same as failing tests — don't suggest the
   change to the user until it passes.

The ` + "`.ops0/config.json`" + ` file binds this repo to an ops0 IaC project.
Org-wide policies always apply; project-specific policies apply when the repo
is bound.
` + end

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return os.WriteFile(path, []byte("# Project notes\n\n"+block+"\n"), 0o644)
	}
	if err != nil {
		return err
	}

	content := string(data)
	beginIdx := indexOf(content, begin)
	endIdx := indexOf(content, end)
	if beginIdx >= 0 && endIdx > beginIdx {
		// Replace existing block in place.
		content = content[:beginIdx] + block + content[endIdx+len(end):]
	} else {
		// Append a new block.
		if len(content) > 0 && content[len(content)-1] != '\n' {
			content += "\n"
		}
		content += "\n" + block + "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// Tiny helper to avoid importing strings just for this; keeps the function
// readable above.
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
