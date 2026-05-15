<div align="center">

# ops0 CLI

**Policy, lint, vulnerability, and cost guardrails for AI coding agents.**

Sits in front of Claude Code, Codex and Gemini CLI. Every time the agent
finishes writing IaC, the whole working directory gets validated, linted,
policy-checked, security-scanned, and cost-estimated server-side. Failures
come back as a failed tool call so the agent self-remediates. Destructive
commands are blocked before they run.

[![Latest Release](https://img.shields.io/github/v/release/ops0-ai/ops0-cli?display_name=tag&sort=semver)](https://github.com/ops0-ai/ops0-cli/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ops0-ai/ops0-cli)](https://goreportcard.com/report/github.com/ops0-ai/ops0-cli)

```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | sh
```

[Quick start](#quick-start)
· [What runs at end-of-turn](#what-runs-at-end-of-turn)
· [How agents trigger it](#how-agents-trigger-it)
· [Monorepos](#monorepos)
· [ops0-scan.md](#the-ops0-scanmd-report)
· [Commands](#commands)

</div>

---

## Why this exists

When humans write infrastructure, policy gates fire on the PR. CI runs the
scanner, someone gets paged, the PR sits in review for hours.

When an **agent** writes infrastructure, that loop is broken. The agent
will happily generate a public S3 bucket, an open security group, or an
oversized EC2 fleet. By the time CI catches it, the agent has moved on.

`ops0` sits in front of the agent. It runs your organization's checks
when the agent finishes a turn, blocks destructive commands before they
execute, tells the agent how much its IaC will cost per month, and gates
the change against the project's budget.

## What it does

| | |
|---|---|
| **Validates at end-of-turn, not per file save** | A `Stop` hook fires when the agent finishes its turn. `ops0 validate` runs once against the complete working directory. Validating mid-construction was noisy on half-written modules; validating the finished module is clean. |
| **Blocks destroy commands** | A `PreToolUse` hook intercepts `terraform destroy`, `tofu destroy`, `oxid destroy` and the `-destroy` variants — before they run. Override with `OPS0_ALLOW_DESTROY=1`. |
| **Enforces project budgets** | If the cost estimate exceeds a project budget set in the ops0 dashboard, the gate fails. Agent gets told to optimize. |
| **Writes a fresh report file** | `ops0-scan.md` is rewritten at the repo root after every run, so the agent reads one file to know the current state across all stages. |
| **Speaks MCP** | `ops0 mcp serve` exposes `list_policies` and `check_compliance` to any MCP-compatible agent. Registered automatically with Claude Code on `ops0 init`. |
| **Multi-project aware** | Walks up *and* down from the workspace root to find the nearest `.ops0/config.json` (up to 12 levels deep). One repo with many subprojects each maps to its own ops0 project. |
| **Audit trail** | Every failed lint finding, policy violation, vulnerability, blocked destroy, and budget overrun is recorded against your API key in `Settings → API Keys → Activity`. |
| **Works from any workspace root** | Hooks install at both project-level and user-level, so they fire no matter where Claude Code is opened. |

## Quick start

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | sh

# 2. Auth: get a key at https://brew.ops0.ai/settings?tab=api-keys
ops0 login --api-base https://brew.ops0.ai

# 3. Bind a repo (or any subdir) to an ops0 project
cd ~/work/my-terraform-repo
ops0 init --project=<project-id>

# 4. Open Claude Code (or Codex / Gemini) anywhere in or above the repo.
#    Write Terraform freely. When the agent stops, the gate fires once
#    against the whole module. Try `terraform destroy` and watch it get
#    blocked before it runs.
```

Verify the wiring:

```console
$ ops0 policies list
NAME                              CATEGORY      SEVERITY  DESCRIPTION
no-public-s3                      security      high      S3 buckets must not be public
require-encryption-at-rest        security      medium    All storage must use customer-managed keys
tag-required-cost-center          tagging       low       Every resource must carry a cost-center tag

$ ops0 validate .
ops0 validate . (4 files, 8.2s)

✓ Configuration is valid

tflint: 0 error(s), 2 warning(s), 0 notice(s)
  [WARNING] terraform_required_providers: Missing version constraint for provider "aws"

scan: 14 passed, 8 failed (0 parsing errors). Severity: 1C / 1H / 6M / 0L
  [CRITICAL] no-public-s3: S3 bucket has public read access
  [HIGH]     require-encryption: S3 bucket is missing default encryption
  ...

cost: $284.50 / month across 6 resource(s)
  $148.92   aws_db_instance.app (aws_db_instance)
  $89.71    aws_instance.api (aws_instance)
  ...

budget: ✓ $284.50/mo within project limit of $500.00/mo.
```

## What runs at end-of-turn

A single `ops0 validate` call fans out to **five** server-side stages, in
this order. The CLI returns the merged result; failures gate the agent.

| # | Stage | Catches | Fails the gate when... |
|---|---|---|---|
| 1 | **Syntax validation** | parse errors, undefined variables, wrong attribute types | `terraform validate` returns invalid |
| 2 | **Lint** (provider-aware) | wrong instance types, deprecated args, missing version constraints | lint errors (warnings/notices report only) |
| 3 | **Policies + vulnerabilities** | your org's compliance rules, security findings (public buckets, open SGs, IMDSv1, unencrypted volumes, missing tags, etc.) | any finding at or above `--scan-fail-on` (default `high`) |
| 4 | **Cost estimate** | monthly cost of all priced resources | informational unless step 5 triggers |
| 5 | **Project budget** | per-project monthly limit from the ops0 dashboard | `enabled` AND `exceeded` AND `Block Deployments on Exceed` is on |

All five run in one HTTPS call. With the server-side provider cache warm,
end-of-turn validation is ~5-12s.

## How agents trigger it

`ops0 init` writes hooks at both project- and user-level `.claude/settings.json`,
so the gate fires regardless of which directory Claude Code is opened at.

```
┌──────────────────────────────────────────────┐
│   Claude Code / Codex / Gemini writes        │
│   .tf / .tofu / .hcl / .tfvars files         │
│   in any order. No mid-turn gate fires.      │
└──────────────────────┬───────────────────────┘
                       │
                       │ Agent finishes its turn
                       ▼
            Stop hook (.claude/settings.json)
                       │
                       ▼
   ┌─────────────────────────────────────────┐
   │   ops0 validate <bound-dir>             │
   │   (walks up AND down from workspace     │
   │    root, up to 12 levels deep, to       │
   │    find the nearest .ops0/config.json)  │
   └─────────────────┬───────────────────────┘
                     │ HTTPS (API key)
                     ▼
   ┌─────────────────────────────────────────┐
   │   ops0 platform                         │
   │   - syntax validate                     │
   │   - lint                                │
   │   - policies + vulnerabilities          │
   │   - cost                                │
   │   - project budget                      │
   └─────────────────┬───────────────────────┘
                     │
                     ▼
   ┌─────────────────────────────────────────┐
   │  exit 0 → turn ends normally            │
   │  exit ≠ 0 → hook fails →                │
   │  agent gets stderr, takes another turn  │
   │  to remediate.                          │
   │                                         │
   │  Either way: ops0-scan.md is rewritten  │
   │  and Activity rows land in the          │
   │  dashboard.                             │
   └─────────────────────────────────────────┘
```

The user never has to ask "did you run the scan?". The gate is mechanical.

## Destructive command blocking

When the agent tries to run `terraform destroy` / `tofu destroy` / `oxid destroy`
(or any `-destroy` variant) via Bash, the `PreToolUse` hook fires **before**
the command runs:

```
agent calls Bash with `terraform destroy -auto-approve`
     │
     ▼  PreToolUse hook reads the command
     ▼  matches the destroy pattern
     ▼  POSTs an audit row to ops0
     ▼  prints the block message to stderr
     ▼  exit 2  →  Claude Code aborts the Bash call
```

To intentionally tear something down (sandbox, dev env), prefix with the
override:

```bash
OPS0_ALLOW_DESTROY=1 terraform destroy
```

The override is still logged to the audit trail.

This hook is separate from the validate pipeline — it's a runtime safety
gate, not a write-time gate.

## Monorepos

One repo, many subprojects, each with its own policies and budget? Open
Claude Code anywhere in or above any of them. The Stop hook resolves the
binding automatically:

```
my-monorepo/                                ← Claude Code opened here
├── iaas/
│   └── terraform/
│       └── live/
│           └── env/
│               └── prod/
│                   └── customer/
│                       ├── alpha/
│                       │   ├── .ops0/config.json    ← projectId: alpha
│                       │   └── main.tf
│                       └── beta/
│                           ├── .ops0/config.json    ← projectId: beta
│                           └── main.tf
└── shared/
    ├── .ops0/config.json                   ← projectId: shared
    └── main.tf
```

How it finds the binding:

1. Walk **up** from the workspace's current directory looking for the
   nearest `.ops0/config.json`. If found, that's the target.
2. Otherwise walk **down** the workspace tree up to **12 levels deep**
   and take the nearest descendant. Pruned automatically:
   `node_modules`, `.git`, `.terraform`, `dist`, `build`, `.next`, `.venv`,
   `venv`. The scan stays fast on workspaces with large vendored trees.

So you can open Claude Code at `~/my-monorepo/` and edit
`iaas/terraform/live/env/prod/customer/alpha/main.tf` — the Stop hook
walks down through 9 levels of directories, finds `alpha/.ops0/config.json`,
and validates against the `alpha` project's policies + budget. Same edit
in `beta/` resolves to the `beta` project independently.

12-level depth covers any realistic IaC layout. If you need deeper, file
an issue with your path shape and we'll bump it.

## The ops0-scan.md report

After every `ops0 validate` run, the CLI rewrites a markdown file at the
bound repo root:

```
<bound-dir>/ops0-scan.md
```

It contains:

- Generated timestamp + CLI version
- Summary table (validate / lint / policies / cost / budget — one row each)
- terraform validate errors, if any
- Lint findings table
- Failed policy + vulnerability findings table, ranked by severity
- Cost breakdown, top 20 resources by monthly cost
- Budget verdict (within limit / over by $X / blocked)

The file is overwritten on every run, so it's always the current truth.
The agent reads it between turns to see findings without re-running
validate. Don't hand-edit it (the next run will throw your changes away).

Disable with `--no-report`, or move it with `--report path/to/file.md`.

## Integrations

### Claude Code

`ops0 init` does the wiring for you:

```bash
ops0 init --project=<project-id>
```

That single command:

- Writes `<cwd>/.ops0/config.json` to bind the directory to a project.
- Installs `PreToolUse` (block destroys) and `Stop` (end-of-turn validate)
  hooks in `<cwd>/.claude/settings.json`.
- Installs the same hooks in `~/.claude/settings.json` so they fire
  whatever directory Claude Code is opened at.
- Appends a fenced governance section to `CLAUDE.md` so the agent knows
  the rules and how to read `ops0-scan.md`.
- Runs `claude mcp add ops0 ops0 mcp serve` so the agent can call
  `list_policies` and `check_compliance` natively over MCP.

After upgrading the CLI, re-run `ops0 init --force` in the project
directory to refresh the hooks, then **restart your Claude Code session**
so it re-reads the hook config.

### Codex / Gemini CLI / any MCP client

```bash
ops0 mcp serve
```

Run it as a stdio MCP server. Tools exposed: `list_policies`,
`check_compliance`, `whoami`. Wire it up via your client's MCP config.

## Commands

| Command                          | What it does                                                          |
|----------------------------------|-----------------------------------------------------------------------|
| `ops0 login`                     | Authenticate with an API key from the ops0 settings UI                |
| `ops0 init`                      | Bind the current directory to a project, install hooks, register MCP  |
| `ops0 policies list`             | List policies in scope for the current directory's project            |
| `ops0 policies check [path]`     | Lightweight scan (policies + vulnerabilities only, no init/lint/cost) |
| `ops0 validate [path]`           | Full pipeline: syntax + lint + policies + vulnerabilities + cost      |
| `ops0 mcp serve`                 | Run the MCP server over stdio                                         |
| `ops0 telemetry blocked-command` | Record a destroy attempt blocked by the PreToolUse hook               |
| `ops0 version`                   | Print version info                                                    |

### `ops0 validate` flags

| Flag | Default | Purpose |
|---|---|---|
| `--format pretty\|json` | `pretty` | Output format. JSON is for piping into other tools. |
| `--iac-type terraform\|opentofu\|oxid` | `terraform` | Which IaC flavor to dispatch to. |
| `--cloud aws\|gcp\|azure\|oracle` | (auto) | Hint for the lint plugins. |
| `--scan-fail-on critical\|high\|medium\|low` | `high` | Severity threshold for the policy/vulnerability gate. |
| `--fail-on-warning` | `false` | Also exit non-zero on lint warnings. |
| `--report <path>` | `<bound-dir>/ops0-scan.md` | Where to write the report. |
| `--no-report` | `false` | Skip writing the report file. |

### `ops0 init` flags

| Flag | Default | Purpose |
|---|---|---|
| `--project <id>` | `""` | ops0 IaC project ID to bind this directory to. Get it from the dashboard. |
| `--force` | `false` | Overwrite an existing `.ops0/config.json` and refresh hook commands. |
| `--skip-claude` | `false` | Don't write `.claude/settings.json` or register the MCP server. Useful in CI. |

## Config files

| Path | Scope | Purpose |
|---|---|---|
| `~/.ops0/config.yaml` | User-wide | Credentials and defaults (`chmod 0600`) |
| `<dir>/.ops0/config.json` | Per-directory | Project binding. Commit this to git. |
| `<dir>/.claude/settings.json` | Per-directory | Project-level Claude Code hooks |
| `~/.claude/settings.json` | User-wide | User-level Claude Code hooks (fire from any workspace) |
| `<dir>/ops0-scan.md` | Per-directory | Auto-generated scan report. Read it; don't edit it. |

## Build from source

```bash
git clone https://github.com/ops0-ai/ops0-cli && cd ops0-cli
go build -o ops0 ./cmd/ops0
sudo install -m 0755 ops0 /usr/local/bin/ops0
```

Requires Go 1.22 or later.

## Contributing

Issues and PRs welcome. Guardrails:

- `go vet ./...` and `go test ./...` before pushing.
- Hook scripts must work on macOS bash 3.2 and Linux bash 4+.
- New telemetry fields need a paired migration in the `config-master` repo.
- Telemetry calls are best-effort. Never block the CLI on a failed report.

## Star history

If this is useful to you, star it. It's the cheapest signal that helps
other teams find it.

## License

Apache 2.0. See [LICENSE](./LICENSE).
