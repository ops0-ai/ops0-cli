<div align="center">

# ops0 CLI

**Policy guardrails for AI coding agents.**

Scans the Terraform, OpenTofu and OCI manifests your AI agent writes.
Blocks `terraform destroy` before it runs. Ships an audit trail to ops0.
Works with Claude Code, Codex and Gemini CLI.

[![Latest Release](https://img.shields.io/github/v/release/ops0-ai/ops0-cli?display_name=tag&sort=semver)](https://github.com/ops0-ai/ops0-cli/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ops0-ai/ops0-cli)](https://goreportcard.com/report/github.com/ops0-ai/ops0-cli)

```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | sh
```

[Quick start](#quick-start)
· [How it works](#how-it-works)
· [Integrations](#integrations)
· [FAQ](#faq)
· [Contributing](#contributing)

</div>

---

## Why this exists

When humans write infrastructure, policy gates fire on the PR. CI runs the
scanner, someone gets paged, the PR sits in review for hours.

When an **agent** writes infrastructure, that loop is broken. The agent
will happily generate a public S3 bucket, an open security group, a wide
IAM policy. By the time CI catches it, the agent has moved on.

`ops0` sits in front of the agent. It reads your organization's policies,
tells the agent the rules before it generates, fails the agent's edit if
the result violates a policy, and blocks destructive commands before they
run. All audited.

> If you're using Claude Code, Codex, or Gemini CLI to ship infrastructure,
> this is the smallest, sanest thing you can put between the model and your
> cloud.

## What it does

| | |
|---|---|
| **Scans IaC after every edit** | A `PostToolUse` hook runs `ops0 policies check` against every `.tf` / `.tofu` / `.hcl` file the agent writes. Violations come back to the model as a failed tool call, so the agent self-remediates. |
| **Blocks destroy commands** | A `PreToolUse` hook intercepts `terraform destroy`, `tofu destroy`, `oxid destroy` and the `-destroy` variants. Override with `OPS0_ALLOW_DESTROY=1`. |
| **Speaks MCP** | `ops0 mcp serve` exposes `list_policies` and `check_compliance` to any MCP-compatible agent. Registered automatically with Claude Code on `ops0 init`. |
| **Multi-project aware** | Walks up from the edited file to find the nearest `.ops0/config.json`. One repo with ten subprojects each maps to its own ops0 project. |
| **Audit trail** | Every blocked destroy and every failing scan is recorded against your API key. Browse it in `Settings → API Keys → Activity`. |
| **Works everywhere** | Hooks install at both project-level and user-level, so they fire no matter which directory Claude Code opens at. |

## Quick start

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | sh

# 2. Auth: get a key at https://brew.ops0.ai/settings?tab=api-keys
ops0 login --api-base https://brew.ops0.ai

# 3. Bind a repo (or any subdir) to an ops0 project
cd ~/work/my-terraform-repo
ops0 init --project=<project-id>

# 4. Open Claude Code in the repo and write some Terraform.
#    The PostToolUse hook scans every .tf file the agent writes.
#    Try `terraform destroy` and watch the PreToolUse hook block it.
```

Verify it's wired up:

```console
$ ops0 policies list
NAME                              CATEGORY      SEVERITY  DESCRIPTION
no-public-s3                      security      high      S3 buckets must not be public
require-encryption-at-rest        security      medium    All storage must use customer-managed keys
tag-required-cost-center          tagging       low       Every resource must carry a cost-center tag
...

$ ops0 policies check .
12 findings (1 critical, 1 high, 10 medium) across 4 files.
```

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
                                │  - Policy engine     │
                                │  - Telemetry / audit │
                                └──────────────────────┘
```

1. **Policies live on the ops0 platform.** Rules are attached to projects
   and groups in the dashboard. The CLI pulls only the enabled ones at
   request time, so disabling a rule in the UI takes effect immediately.
2. **Scans happen server-side.** The CLI bundles your `.tf` / `.tofu`
   files and sends them over HTTPS. Files are held in a tempdir for the
   duration of the scan and never persisted.
3. **The agent gets the truth.** A failing scan exits non-zero from the
   `PostToolUse` hook with stderr containing the violation. Claude Code,
   Codex, and Gemini CLI all surface that as a failed tool call back to
   the model.
4. **Destroy commands are blocked before they run.** The `PreToolUse` hook
   matches on the Bash command string and exits 2 if it sees a destroy
   pattern. The model sees the block and explains it to you instead of
   tearing down infrastructure.
5. **Everything is audited.** Each failed scan and each blocked destroy
   becomes a row attached to your API key, visible in the ops0 dashboard.

## Integrations

### Claude Code

`ops0 init` does the wiring for you:

```bash
ops0 init --project=<project-id>
```

That single command:

- Writes `<cwd>/.ops0/config.json` to bind the directory to a project.
- Installs `PostToolUse` (scan IaC) + `PreToolUse` (block destroys) hooks
  in `<cwd>/.claude/settings.json`.
- Installs the **same** hooks in `~/.claude/settings.json` so they fire
  whatever directory you open Claude Code at. The user-level hook is
  gated on a `.ops0/config.json` walk-up so unrelated repos aren't
  scanned.
- Appends a fenced governance section to `CLAUDE.md` so the agent reads
  the rules before generating IaC.
- Runs `claude mcp add ops0 ops0 mcp serve` so the agent can call
  `list_policies` and `check_compliance` natively.

Re-run `ops0 init --force` after upgrading the CLI to refresh the hook
scripts.

### Codex / Gemini CLI / any MCP client

```bash
ops0 mcp serve
```

Run that as a stdio MCP server. Tools exposed: `list_policies`,
`check_compliance`, `whoami`. Wire it up via your client's MCP config.

## Multi-project monorepos

One repo, ten subprojects, each with its own policies? That works:

```
my-monorepo/
├── prod/
│   ├── .ops0/config.json     ← projectId: prod
│   └── main.tf
├── staging/
│   ├── .ops0/config.json     ← projectId: staging
│   └── main.tf
└── shared/
    ├── .ops0/config.json     ← projectId: shared
    └── main.tf
```

When the hook fires on `prod/main.tf`, the CLI walks up from that file's
directory to find `prod/.ops0/config.json` and scans against the `prod`
project's policy set. Same edit in `staging/main.tf` resolves to the
`staging` project. You can open Claude Code at `my-monorepo/` (the parent)
and the routing still works.

## Commands

| Command                          | What it does                                                          |
|----------------------------------|-----------------------------------------------------------------------|
| `ops0 login`                     | Authenticate with an API key from the ops0 settings UI                |
| `ops0 init`                      | Bind the current directory to a project, install hooks, register MCP  |
| `ops0 policies list`             | List policies in scope for the current directory's project            |
| `ops0 policies check [path]`     | Scan IaC files at the given path against your policies                |
| `ops0 mcp serve`                 | Run the MCP server over stdio (for Claude Code et al.)                |
| `ops0 telemetry blocked-command` | Record a destroy attempt blocked by the PreToolUse hook               |
| `ops0 version`                   | Print version info                                                    |

## Config files

| Path | Scope | Purpose |
|---|---|---|
| `~/.ops0/config.yaml` | User-wide | Credentials and defaults (`chmod 0600`) |
| `<dir>/.ops0/config.json` | Per-directory | Project binding. Commit this to git. |
| `<dir>/.claude/settings.json` | Per-directory | Project-level Claude Code hooks |
| `~/.claude/settings.json` | User-wide | User-level Claude Code hooks (fire from any workspace) |

## FAQ

**Does it send my Terraform to the cloud?**
Yes, over HTTPS, scoped by your API key. Files live in a tempdir on the
scanner pod for the duration of the scan and are not persisted.

**What happens if my CI runs `terraform apply`?**
`apply` is intentionally not blocked. Blocking every `apply` would be
unworkable. The right defense for `apply` is plan-aware, and that's on
the roadmap. Today the focus is preventing the agent from writing bad
IaC in the first place and preventing it from tearing down what's there.

**How do I let `terraform destroy` through for a planned tear-down?**
Prefix the command with `OPS0_ALLOW_DESTROY=1`. The block still gets
logged to the audit trail.

**I deleted `.ops0/config.json`. What happens?**
The user-level hook walks up and finds nothing, so it exits 0. The
directory is unbound. Nothing weird happens.

**I have ten projects in one repo. Will the hook know which one?**
Yes. The CLI walks up from the edited file to find the nearest
`.ops0/config.json` and uses that project's policies.

**Does this work with anything other than Terraform?**
Today: Terraform, OpenTofu, and OCI manifests via the `.tf`, `.tofu`,
`.hcl`, and `.tf.json` extensions. Kubernetes manifests are next.

## Build from source

```bash
git clone https://github.com/ops0-ai/ops0-cli && cd ops0-cli
go build -o ops0 ./cmd/ops0
sudo install -m 0755 ops0 /usr/local/bin/ops0
```

Requires Go 1.22 or later. The binary statically links everything except
glibc (Linux) and the system Python that the hook scripts call for JSON
parsing.

## Contributing

Issues and PRs welcome. A few guardrails:

- Run `go vet ./...` and `go test ./...` before pushing.
- Keep the CLI tree-shakeable: any new dependency must justify its place.
- Hook scripts have to work on macOS bash 3.2 and Linux bash 4+. No
  bashisms beyond what's already in `internal/cmd/init.go`.
- New telemetry fields require a paired migration in the `config-master`
  repo. Best-effort writes only; never block the CLI on telemetry.

## Star history

If this is useful to you, star it. It's the cheapest signal that helps
other teams find it.

## License

Apache 2.0. See [LICENSE](./LICENSE).
