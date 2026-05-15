package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	initProjectID string
	initForce     bool
	initSkipClaude bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Wire up this repository for ops0 governance",
	Long: `Initializes the current repository for ops0 governance. Three things happen:

  .ops0/config.json             Per-repo binding (project ID, paths, policy version).
                                Check this into git so collaborators inherit the same
                                policy set.
  CLAUDE.md                     A fenced governance section is appended (or refreshed)
                                instructing agents to call ops0 before generating IaC.
  .claude/settings.json         A Stop hook is installed. When the agent finishes a turn,
                                'ops0 validate' runs against the working directory and
                                fails non-zero on any policy / lint / cost / budget
                                violation, forcing the agent to remediate. A PreToolUse
                                hook blocks 'terraform destroy' / 'tofu destroy' /
                                'oxid destroy' before they run.
  Claude Code MCP registration  Best-effort: if 'claude' is on PATH we register the
                                ops0 MCP server so the agent can call list_policies
                                and check_compliance natively. Skip with --skip-claude.

Idempotent: re-running replaces fenced sections without clobbering your edits.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initProjectID, "project", "", "ops0 IaC project ID to bind this repo to")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite an existing .ops0/config.json")
	initCmd.Flags().BoolVar(&initSkipClaude, "skip-claude", false, "Don't register MCP server / write Claude Code hooks (still writes .ops0/ and CLAUDE.md)")
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

	repoCfg := &config.RepoConfig{ProjectID: initProjectID}
	if err := config.SaveRepo(cwd, repoCfg); err != nil {
		return fmt.Errorf("write repo config: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Wrote %s\n", config.RepoConfigPath(cwd))

	if err := upsertClaudeMd(cwd); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update CLAUDE.md: %v\n", err)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Updated CLAUDE.md governance block")
	}

	if !initSkipClaude {
		if err := upsertClaudeHooks(cwd); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not install Claude Code hooks: %v\n", err)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Installed .claude/settings.json hooks (Stop + PreToolUse)")
		}

		// User-level hooks: same destroy block + gated policy check.
		// Fires for every CC session regardless of which directory CC opened,
		// so editing this repo's IaC from a parent workspace still triggers
		// the scan. Gated on `.ops0/config.json` walk-up so unrelated repos
		// don't get policy checks they didn't opt into.
		if err := upsertUserClaudeHooks(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not install user-level Claude Code hooks: %v\n", err)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Installed ~/.claude/settings.json hooks (fire from any workspace)")
		}

		if err := registerClaudeMCP(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not register MCP server: %v\n", err)
			fmt.Fprintln(cmd.ErrOrStderr(), "         You can run it manually: claude mcp add ops0 -- ops0 mcp serve")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Registered ops0 MCP server with Claude Code")
		}
	}

	if initProjectID == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "\nNote: no --project bound. Only org-wide policies will apply.")
		fmt.Fprintln(cmd.OutOrStdout(), "Re-run with --project=<id> --force to bind to a specific IaC project.")
	}
	return nil
}

// ─── CLAUDE.md governance block ────────────────────────────────────────────

func upsertClaudeMd(repoRoot string) error {
	path := filepath.Join(repoRoot, "CLAUDE.md")
	const (
		begin = "<!-- ops0:begin -->"
		end   = "<!-- ops0:end -->"
	)
	block := begin + `
## ops0 Governance (auto-managed by ` + "`ops0 init`" + `)

This repository is governed by [ops0](https://brew.ops0.ai). The gate fires
**once per turn**, when you finish writing — not on every individual file
save. Write the whole module in whatever order makes sense, declare
variables / outputs / resources however you like, then stop. The
` + "`Stop`" + ` hook in ` + "`.claude/settings.json`" + ` runs
` + "`ops0 validate`" + ` against the complete working directory:

  syntax validation → lint → policy + vulnerability scan → cost estimate →
  project budget check

Any failure exits non-zero and the violations are surfaced back to you.
**Treat that the same as a failing test** — fix the issues and try again.
The hook runs again on your next turn end.

Other things to know:

1. Call ` + "`list_policies`" + ` on the ops0 MCP server (or run
   ` + "`ops0 policies list`" + ` in a shell) to see the rules that apply
   before generating IaC. Reading the rules first usually means fewer
   round trips through the gate.
2. Destructive Bash commands (` + "`terraform destroy`" + `, ` + "`tofu destroy`" + `,
   ` + "`oxid destroy`" + `) are blocked by the ` + "`PreToolUse`" + ` hook
   before they run. This is a runtime safety gate, not part of validate.
   To intentionally tear down a sandbox, prefix the command with
   ` + "`OPS0_ALLOW_DESTROY=1`" + `.
3. After every validate run, the consolidated report is rewritten at
   ` + "`ops0-scan.md`" + ` at the repo root. Read this file to see the
   current state of findings without re-running validate yourself. Do
   NOT hand-edit it — it's regenerated on every turn.

The ` + "`.ops0/config.json`" + ` file binds this repo to an ops0 project.
Org-wide policies always apply; project-specific policies apply when the repo
is bound. Project-level monthly budgets, when configured in the ops0
dashboard, gate the agent if a change would push monthly cost over the limit.
` + end

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return os.WriteFile(path, []byte("# Project notes\n\n"+block+"\n"), 0o644)
	}
	if err != nil {
		return err
	}

	content := string(data)
	beginIdx := strings.Index(content, begin)
	endIdx := strings.Index(content, end)
	if beginIdx >= 0 && endIdx > beginIdx {
		content = content[:beginIdx] + block + content[endIdx+len(end):]
	} else {
		if len(content) > 0 && content[len(content)-1] != '\n' {
			content += "\n"
		}
		content += "\n" + block + "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// ─── Claude Code hooks ─────────────────────────────────────────────────────

// upsertClaudeHooks installs a PostToolUse hook in <repo>/.claude/settings.json
// that fires after Claude Code writes or edits an IaC file. The hook runs
// `ops0 policies check <file>` and surfaces violations back to the agent.
//
// Claude Code merges per-repo settings on top of user settings, so writing
// at the repo level is the least invasive thing we can do — it doesn't
// touch the user's ~/.claude/settings.json and stays scoped to this repo.
//
// File format (Claude Code settings.json hooks schema):
//
//   {
//     "hooks": {
//       "PostToolUse": [
//         {
//           "matcher": "Edit|Write|MultiEdit",
//           "hooks": [
//             { "type": "command", "command": "..." }
//           ]
//         }
//       ]
//     }
//   }
func upsertClaudeHooks(repoRoot string) error {
	dir := filepath.Join(repoRoot, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeClaudeHooks(filepath.Join(dir, "settings.json"), preToolUseCmd, stopHookCmd)
}

// upsertUserClaudeHooks writes the same destroy block and IaC policy check to
// the user-level ~/.claude/settings.json. Because Claude Code's project-level
// settings only fire when CC is launched at that workspace root, this is what
// makes the hooks robust to "user opened a parent directory in CC."
//
// The user-level PostToolUse hook is GATED: it walks up from the edited
// file's directory looking for a `.ops0/config.json`. Only if it finds one
// (meaning `ops0 init` was run somewhere up the tree) does it invoke the
// scanner. That way unrelated repos don't get policy checks they didn't
// opt into.
//
// The PreToolUse destroy block is NOT gated — destroy is dangerous in any
// repo, and the `OPS0_ALLOW_DESTROY=1` escape hatch still works.
func upsertUserClaudeHooks() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeClaudeHooks(filepath.Join(dir, "settings.json"), preToolUseCmd, stopHookCmd)
}

// ── Stop hook: end-of-turn full validate ─────────────────────────────────────
//
// Fires when Claude finishes its response. We run the heavy pipeline
// (init + validate + tflint) once per turn rather than per file edit —
// init can take 5-30s for provider download, so per-edit would be brutal.
//
// Gating: only fire if a .ops0/config.json exists in CWD's ancestry. If the
// workspace isn't ops0-bound we exit 0 immediately. This avoids running
// validation on every Claude turn in unrelated repos.
const stopHookCmd = `# ops0 Stop hook — end-of-turn IaC validate + tflint.
# Walk up from CWD looking for .ops0/config.json. If not found, no-op.
d="$PWD"
found=""
while [ "$d" != "/" ] && [ "$d" != "." ] && [ -n "$d" ]; do
  if [ -f "$d/.ops0/config.json" ]; then found="$d"; break; fi
  d="$(dirname "$d")"
done
if [ -z "$found" ]; then exit 0; fi
# Only run if .tf/.tofu/.hcl files in the bound dir were touched in the
# last 5 minutes — proxy for "the agent worked on IaC this turn."
if ! find "$found" -type f \( -name "*.tf" -o -name "*.tofu" -o -name "*.hcl" -o -name "*.tf.json" \) -mmin -5 2>/dev/null | grep -q .; then
  exit 0
fi
# Run the validation. Non-zero exit surfaces stderr to the model so it
# self-remediates the failure.
ops0 validate "$found" 1>&2 || exit 2
`

// ── Hook command strings ─────────────────────────────────────────────────────
//
// As of v0.5.21 we install only two hooks: PreToolUse (destroy block) and
// Stop (end-of-turn full validate). The PostToolUse per-edit gate was
// removed because validating a half-written module produced noise and
// imposed unnatural authoring order on the agent. Validation now runs
// once, against the complete module, when the agent finishes its turn.

// ── PreToolUse: block destructive IaC commands BEFORE they run ───────────
//
// Fires on the Bash tool. We extract `tool_input.command` and pattern-match
// against `terraform/tofu/oxid destroy` (and `terraform plan -destroy`).
// PreToolUse is the right phase for command blocking — PostToolUse fires
// AFTER the command runs, by which point your infra is gone.
//
// Escape hatch: `OPS0_ALLOW_DESTROY=1` env var bypasses the block. Useful
// for planned tear-downs of dev environments without relaxing the policy.
//
// `terraform apply` is intentionally NOT blocked — applies are routine and
// blocking all of them would be unworkable. The right defense for apply
// is plan-aware: see `ops0 plan check` (coming next iteration).
// On a destroy match we do two things in this order:
//   1) Best-effort: `ops0 telemetry blocked-command` POSTs the audit row
//      so the Activity tab in Settings shows the attempt. This call
//      ALWAYS exits 0 internally so its result can't suppress the block.
//   2) Print the human-readable block message and exit 2.
// The telemetry hop is wrapped in `>/dev/null 2>&1 &` so even if the
// network is hung the hook returns within a few hundred ms.
const preToolUseCmd = `if [ -n "${OPS0_ALLOW_DESTROY:-}" ]; then exit 0; fi
cmd="$(python3 -c 'import json,sys; print((json.load(sys.stdin).get("tool_input") or {}).get("command",""))')"
case "$cmd" in
  *"terraform destroy"*|*"tofu destroy"*|*"oxid destroy"*|*"terraform plan -destroy"*|*"tofu plan -destroy"*)
    ops0 telemetry blocked-command "$cmd" --pattern "destroy" --title "Destructive IaC command blocked" >/dev/null 2>&1 || true
    echo "ops0 governance: this command would destroy infrastructure." 1>&2
    echo "  Command: $cmd" 1>&2
    echo "  Blocked by: organization policy (no unrestricted destroy)" 1>&2
    echo "  Override:   prefix with  OPS0_ALLOW_DESTROY=1  and rerun." 1>&2
    echo "  Recorded:   visible in Settings → API Keys → Activity" 1>&2
    exit 2
    ;;
esac`

// writeClaudeHooks merges ops0 PreToolUse + Stop hooks into an arbitrary
// Claude Code settings.json (project-level or user-level). Used by both
// upsertClaudeHooks (repo's .claude/settings.json) and
// upsertUserClaudeHooks (~/.claude/settings.json).
//
// As of v0.5.21 we no longer install a PostToolUse hook. Per-edit gating
// on a half-written module was noisy and forced unnatural authoring order
// on the agent. Validation now runs once per agent turn via the Stop
// hook, against the complete working directory.
//
// The `_ops0: true` sentinel on each entry lets us re-run init without
// accumulating duplicate hooks. For PostToolUse specifically we STRIP
// any prior `_ops0` entry without re-adding one, so users who upgrade
// via `ops0 init --force` lose the old per-edit gate cleanly.
//
// PreToolUse — block destroy commands (runtime safety, separate concern)
// Stop       — end-of-turn full pipeline (validate, lint, scan, cost, budget)
func writeClaudeHooks(path, preCmd, stopCmd string) error {
	var settings map[string]any
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = map[string]any{}
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}

	// removeOps0Entries strips every prior `_ops0` entry from the named
	// event. Used to garbage-collect the old PostToolUse hook on upgrade.
	removeOps0Entries := func(event string) {
		arr, _ := hooks[event].([]any)
		filtered := make([]any, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if ok && m["_ops0"] == true {
				continue
			}
			filtered = append(filtered, item)
		}
		if len(filtered) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = filtered
		}
	}

	// upsertEntry replaces any prior `_ops0` entry on this event with our
	// fresh one. Stop has no matcher; PreToolUse uses Bash.
	upsertEntry := func(event string, matcher string, cmd string) {
		arr, _ := hooks[event].([]any)
		filtered := make([]any, 0, len(arr)+1)
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if ok && m["_ops0"] == true {
				continue
			}
			filtered = append(filtered, item)
		}
		entry := map[string]any{
			"_ops0": true,
			"hooks": []any{
				map[string]any{"type": "command", "command": cmd},
			},
		}
		if matcher != "" {
			entry["matcher"] = matcher
		}
		filtered = append(filtered, entry)
		hooks[event] = filtered
	}

	// On upgrade: drop the old PostToolUse hook entirely. We do NOT
	// re-add it — Stop is the new (and only) validation gate.
	removeOps0Entries("PostToolUse")

	upsertEntry("PreToolUse", "Bash", preCmd)
	upsertEntry("Stop", "", stopCmd)
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// ─── Claude Code MCP server registration ───────────────────────────────────

// registerClaudeMCP tries to register the ops0 MCP server with Claude Code's
// user-scope configuration. We invoke `claude mcp add` rather than writing
// the config file directly — that way we stay in sync with whatever path /
// schema Claude Code is using on this version.
//
// If the `claude` binary isn't on PATH we soft-fail with a hint. If the
// server is already registered we treat that as success.
func registerClaudeMCP() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not on PATH; install Claude Code or skip with --skip-claude")
	}

	// Resolve our own absolute path so the MCP entry doesn't depend on PATH
	// at the time Claude Code spawns us.
	self, err := os.Executable()
	if err != nil {
		self = "ops0" // best-effort fallback
	}

	// `claude mcp add <name> <command> [args...]` — args after `--` are
	// passed verbatim to the spawned process.
	cmd := exec.Command("claude", "mcp", "add", "ops0", self, "mcp", "serve")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Claude Code returns non-zero if the server already exists; treat
		// "already exists" as success since the contract is idempotent setup.
		txt := strings.ToLower(string(out))
		if strings.Contains(txt, "already") {
			return nil
		}
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
