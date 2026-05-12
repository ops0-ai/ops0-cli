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

### Also install OPA (one-time)

The CLI shells out to the [Open Policy Agent](https://www.openpolicyagent.org/)
binary for local Rego evaluation. Code never leaves your machine.

```bash
brew install opa            # macOS
# or download from https://www.openpolicyagent.org/docs/latest/#running-opa
```

## Quick start

```bash
# 1. Generate an API key at https://brew.ops0.ai/settings → API Keys → New API Key
ops0 login
# paste your key when prompted

# 2. In your IaC repo, bind it to an ops0 project (optional but recommended)
cd ~/work/my-terraform-repo
ops0 init --project=<project-id>

# 3. See which policies apply
ops0 policies list

# 4. Check your IaC locally — code never leaves your machine
terraform plan -out=plan.out
terraform show -json plan.out > plan.json
ops0 policies check plan.json
```

## Integrate with Claude Code

Add this to your Claude Code MCP config (`~/.config/claude/mcp.json`):

```json
{
  "mcpServers": {
    "ops0": {
      "command": "ops0",
      "args": ["mcp", "serve"]
    }
  }
}
```

Now Claude Code can call `list_policies` and `check_compliance` while it
writes Terraform for you. Combined with `ops0 init`, which appends a
governance section to your `CLAUDE.md`, the agent will:

1. Read the rules before generating any IaC
2. Self-check before suggesting the edit
3. Refuse to propose code that fails a policy

## How it works

```
┌─────────────────────┐         ┌──────────────────────┐
│  Claude Code /      │ ──MCP── │  ops0 CLI (local)    │
│  Codex / Gemini CLI │         │  - OPA local eval    │
└─────────────────────┘         │  - API client        │
                                └──────────┬───────────┘
                                           │ HTTPS (API key)
                                           ▼
                                ┌──────────────────────┐
                                │  ops0 platform       │
                                │  - Policy storage    │
                                │  - Telemetry / audit │
                                └──────────────────────┘
```

- **Policies live on the ops0 platform** (Rego + metadata, attached to
  projects and groups). The CLI pulls them on demand.
- **Rego evaluation runs locally** via the `opa` binary. Your `.tf` files,
  plan JSONs, and module source never leave your machine.
- **Only check results** (counts, policy IDs, severity, anonymized repo
  hash) are reported back, and only if telemetry is enabled.

## Commands

| Command                | What it does                                                    |
|------------------------|-----------------------------------------------------------------|
| `ops0 login`           | Authenticate with an API key from the ops0 settings UI          |
| `ops0 init`            | Bind the current repo to a project; write a `CLAUDE.md` section |
| `ops0 policies list`   | List policies in scope                                          |
| `ops0 policies check`  | Run OPA locally against a Terraform plan JSON                   |
| `ops0 mcp serve`       | Run the MCP server over stdio (for Claude Code et al.)          |
| `ops0 version`         | Print version info                                              |

## Config files

- `~/.ops0/config.yaml` — user-wide credentials and defaults (`chmod 0600`)
- `<repo>/.ops0/config.json` — per-repo project binding (check this into git)

## Homebrew (coming soon)

We'll publish to a Homebrew tap once the project stabilizes. Until then the
curl installer above is the supported path.

## License

Apache 2.0. See [LICENSE](./LICENSE).
