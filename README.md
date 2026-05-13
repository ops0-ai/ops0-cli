# ops0 CLI

Governance for AI coding assistants. Connects Claude Code, Codex, Gemini CLI,
and any MCP-compatible agent to your organization's [ops0](https://brew.ops0.ai)
policies so the IaC they generate is compliant **before it lands in a PR**.

> Free and open source. Apache-2.0.

## Why

Today, policy enforcement on IaC happens at the deploy gate — `terraform plan`
runs against OPA, and bad code gets blocked after the engineer has already
written it, opened a PR, and waited on CI.

When an AI agent writes the code, you can do better: tell the agent the rules
upfront, and have it generate compliant code in the first place. That's what
this CLI does.

## Install

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | sh
```

That's it. The script detects your OS/arch, pulls the latest release tarball
from GitHub, and drops the binary at `/usr/local/bin/ops0`.

### Windows

Download the latest `.zip` from the
[releases page](https://github.com/ops0-ai/ops0-cli/releases/latest)
and unzip `ops0.exe` somewhere on your `%PATH%`.

### Build from source

```bash
git clone https://github.com/ops0-ai/ops0-cli && cd ops0-cli
go build -o ops0 ./cmd/ops0
sudo install -m 0755 ops0 /usr/local/bin/ops0
```

## Quick start

```bash
# 1. Generate an API key at https://brew.ops0.ai/settings?tab=api-keys
ops0 login --api-base https://brew.ops0.ai
# paste your key when prompted (the --api-base is remembered for next time)

# 2. In your IaC repo (or subdirectory), bind it to an ops0 project
cd ~/work/my-terraform-repo
ops0 init --project=<project-id>

# 3. See which policies apply (walks up to find the nearest .ops0/config.json)
ops0 policies list

# 4. Scan your IaC. Files are sent over HTTPS to ops0, evaluated by Checkov
#    plus your org's Rego policies, and unified findings come back.
ops0 policies check .
```

## Integrate with Claude Code

`ops0 init` does the wiring for you. There is no manual MCP config to maintain:

- Runs `claude mcp add ops0 ops0 mcp serve` so Claude Code can call
  `list_policies` and `check_compliance` while it writes Terraform.
- Installs `PostToolUse` and `PreToolUse` hooks in **both**
  `<dir>/.claude/settings.json` and `~/.claude/settings.json`. The user-level
  file is what makes the hooks fire even when you open Claude Code at a
  parent directory of an ops0-bound repo.
- Appends a fenced governance section to `CLAUDE.md` so the agent reads the
  rules before generating IaC.

With this in place, the agent:

1. Reads the rules in `CLAUDE.md` before generating any IaC.
2. Gets a non-zero `PostToolUse` exit when it writes a non-compliant
   `.tf` / `.tofu` / `.hcl` file, surfacing the violation back to the model
   so it can remediate.
3. Gets blocked before running `terraform destroy`, `tofu destroy`, or
   `oxid destroy`. Override for planned tear-downs with
   `OPS0_ALLOW_DESTROY=1`.

## How it works

```
┌─────────────────────┐         ┌──────────────────────┐
│  Claude Code /      │ ──MCP── │  ops0 CLI (local)    │
│  Codex / Gemini CLI │         │  - HTTPS API client  │
│                     │ ─hook── │  - PostToolUse /     │
└─────────────────────┘         │    PreToolUse hooks  │
                                └──────────┬───────────┘
                                           │ HTTPS (API key)
                                           ▼
                                ┌──────────────────────┐
                                │  ops0 platform       │
                                │  - Policy storage    │
                                │  - Checkov + Rego    │
                                │  - Telemetry / audit │
                                └──────────────────────┘
```

- **Policies live on the ops0 platform.** Rego rules and Checkov rules
  attached to projects and groups. The CLI pulls only those currently
  enabled (disabled policies and disabled groups are filtered out
  server-side).
- **IaC scanning is API-driven.** The CLI bundles your `.tf` / `.tofu`
  files and sends them over HTTPS to ops0's scanner, which runs Checkov
  plus your Rego policies and returns unified findings. Files are held in
  a tempdir for the duration of the scan and never persisted.
- **Monorepo aware.** When the hook fires on a file edit, `policies check`
  walks up from the file's directory to find the nearest `.ops0/config.json`.
  Each subproject in a multi-project repo resolves to its own project ID.
- **Only check results** (counts, policy IDs, severity, file path, line,
  resource, an anonymized repo hash) are reported back for audit
  telemetry, and only if telemetry is enabled.

## Commands

| Command                          | What it does                                                          |
|----------------------------------|-----------------------------------------------------------------------|
| `ops0 login`                     | Authenticate with an API key from the ops0 settings UI                |
| `ops0 init`                      | Bind the current directory to a project; install hooks; register MCP  |
| `ops0 policies list`             | List policies in scope for the current directory's project            |
| `ops0 policies check`            | Scan IaC files at the given path against Checkov + your Rego policies |
| `ops0 mcp serve`                 | Run the MCP server over stdio (for Claude Code et al.)                |
| `ops0 telemetry blocked-command` | Record a destroy attempt blocked by the PreToolUse hook               |
| `ops0 version`                   | Print version info                                                    |

## Config files

- `~/.ops0/config.yaml` — user-wide credentials and defaults (`chmod 0600`)
- `<dir>/.ops0/config.json` — per-directory project binding (commit this to git)
- `<dir>/.claude/settings.json` — project-level Claude Code hooks
- `~/.claude/settings.json` — user-level Claude Code hooks (so they fire from any workspace)

## Homebrew (coming soon)

We'll publish to a Homebrew tap once the project stabilizes. Until then the
curl installer above is the supported path.

## What `ops0 init` actually does

| Action | File / Side effect |
|--------|--------------------|
| Binds the directory to an ops0 IaC project | `<cwd>/.ops0/config.json` |
| Adds a governance section for Claude Code and other agents | `<cwd>/CLAUDE.md` (idempotent, fenced) |
| Installs `PostToolUse` (IaC scan) + `PreToolUse` (block destroys) hooks | `<cwd>/.claude/settings.json` |
| Same hooks at user-level so they fire from any workspace Claude Code opens | `~/.claude/settings.json` (gated on `.ops0/config.json` walk-up so unrelated repos aren't scanned) |
| Registers ops0 as an MCP server with Claude Code | `claude mcp add ops0 …` (best-effort; skips with `--skip-claude`) |

The hook is what gives you actual enforcement: if Claude Code writes a
non-compliant `.tf` file, the hook fails non-zero and the violation is
surfaced back to the model so it can remediate before continuing.

## License

Apache 2.0. See [LICENSE](./LICENSE).
