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
  .claude/settings.json         A PostToolUse hook is installed for .tf / .tofu / .hcl
                                Edit/Write events. After the agent writes IaC, the hook
                                runs 'ops0 policies check <file>' and fails non-zero if
                                any violation is found, forcing the agent to remediate.
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
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Installed .claude/settings.json PostToolUse hook for IaC files")
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

This repository is governed by [ops0](https://brew.ops0.ai) policies. **Before
generating or modifying Infrastructure-as-Code (Terraform / OpenTofu / Oxid)
in this repository:**

1. Call ` + "`list_policies`" + ` on the ops0 MCP server (or run
   ` + "`ops0 policies list`" + ` in a shell) to see the rules that apply.
2. After writing any ` + "`.tf` / `.tofu` / `.hcl`" + ` file, the PostToolUse
   hook in ` + "`.claude/settings.json`" + ` will automatically run
   ` + "`ops0 policies check`" + ` against the file. If it fails, **treat that
   the same as a failing test** — fix the violation before considering the
   task complete.
3. The check uses both ops0's policy library (your org's Rego rules) and
   Checkov security rules. A failing check blocks the suggestion.

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
	return writeClaudeHooks(filepath.Join(dir, "settings.json"), postToolUseProjectCmd, preToolUseCmd, stopHookCmd)
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
	return writeClaudeHooks(filepath.Join(dir, "settings.json"), postToolUseUserCmd, preToolUseCmd, stopHookCmd)
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
// Claude Code's PostToolUse hook passes a JSON payload on STDIN with
// `tool_input.file_path` (Edit/Write) and `tool_response.filePath`
// (newer schema variant). There's no $CLAUDE_FILE_PATHS env var
// despite what older docs suggest — we parse the JSON via python3
// (universal on macOS/Linux, no extra brew install required).
//
// PostToolUse now runs `ops0 validate <bound-dir>` rather than a
// lightweight scan. That gives us four things for free:
//   1. terraform validate parses HCL — so syntax breaks (missing `=`,
//      mismatched braces) exit non-zero immediately. The previous
//      scan-only flow silently returned 0 findings on unparseable input.
//   2. tflint runs — catches provider-aware issues (wrong instance types,
//      deprecated args, missing version constraints).
//   3. The org's security policies still run server-side as part of the
//      pipeline.
//   4. Findings get persisted to the audit trail via /telemetry/validate.
//
// File extensions covered: .tf, .tofu, .hcl, .tf.json, .tfvars,
// .tfvars.json. .tfvars are variable inputs to terraform validate.

// Project-level: when an IaC file is written, validate the whole bound
// directory. The repo root for project-level hooks is by definition the
// CWD where `ops0 init` ran, so we just call `ops0 validate` with no
// argument and let it walk up from the file's dir to find .ops0/config.json.
const postToolUseProjectCmd = `f="$(python3 -c 'import json,sys; d=json.load(sys.stdin); ti=d.get("tool_input") or {}; tr=d.get("tool_response") or {}; print(ti.get("file_path") or tr.get("filePath") or "")')" ; case "$f" in *.tf|*.tofu|*.hcl|*.tf.json|*.tfvars|*.tfvars.json) ops0 validate "$f" 1>&2 || exit 2 ;; esac`

// User-level variant. Walks up from the file's dir looking for
// `.ops0/config.json` so we only validate ops0-bound repos. If the file
// isn't IaC or no binding is found, exit 0 (no-op).
const postToolUseUserCmd = `f="$(python3 -c 'import json,sys; d=json.load(sys.stdin); ti=d.get("tool_input") or {}; tr=d.get("tool_response") or {}; print(ti.get("file_path") or tr.get("filePath") or "")')" ; case "$f" in *.tf|*.tofu|*.hcl|*.tf.json|*.tfvars|*.tfvars.json) d="$(dirname "$f")"; while [ "$d" != "/" ] && [ "$d" != "." ] && [ -n "$d" ]; do if [ -f "$d/.ops0/config.json" ]; then ops0 validate "$d" 1>&2 || exit 2; exit 0; fi; d="$(dirname "$d")"; done ;; esac`

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

// writeClaudeHooks merges ops0 PostToolUse / PreToolUse / Stop hooks into
// an arbitrary Claude Code settings.json (project-level or user-level).
// Used by both upsertClaudeHooks (repo's .claude/settings.json) and
// upsertUserClaudeHooks (~/.claude/settings.json).
//
// The `_ops0: true` sentinel on each entry lets us re-run init without
// accumulating duplicate hook entries — we strip our prior entries and
// re-append.
//
// PostToolUse — per-edit lightweight scan
// PreToolUse  — block destroy commands
// Stop        — end-of-turn init+validate+tflint
func writeClaudeHooks(path, postCmd, preCmd, stopCmd string) error {
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

	// Stop events have no `matcher` field — they fire on every turn end.
	// PostToolUse / PreToolUse use a matcher to scope by tool name.
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

	upsertEntry("PostToolUse", "Edit|Write|MultiEdit", postCmd)
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
