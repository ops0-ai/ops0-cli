package cmd

import (
	"fmt"
	"strings"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/spf13/cobra"
)

// printCheckResult renders a CheckResponse for humans. Plain text only — no
// ANSI dependencies so it's safe to embed in CI logs and Windows terminals.
// Coloring can come later behind a flag.
func printCheckResult(cmd *cobra.Command, r *api.CheckResponse, target string, fileCount int) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Target: %s  (%d file%s scanned)\n", target, fileCount, plural(fileCount))
	fmt.Fprintf(
		out,
		"Findings: %d passed  •  %d failed  •  %d skipped\n",
		r.Summary.Passed, r.Summary.Failed, r.Summary.Skipped,
	)

	sd := r.SeverityDistribution
	if r.Summary.Failed > 0 {
		fmt.Fprintf(
			out,
			"By severity:  %d critical  •  %d high  •  %d medium  •  %d low\n\n",
			sd.Critical, sd.High, sd.Medium, sd.Low,
		)
	} else {
		fmt.Fprintln(out)
	}

	if r.Summary.Failed == 0 {
		fmt.Fprintln(out, "✓ No violations.")
		return
	}

	// Print only failed findings, sorted by severity desc so the worst stuff
	// is on top. Same rank order as shouldExitNonZero in policies.go.
	severityRank := func(s string) int {
		switch strings.ToLower(s) {
		case "critical":
			return 4
		case "high":
			return 3
		case "medium":
			return 2
		case "low":
			return 1
		}
		return 0
	}

	// Stable insertion sort — typical scan has <50 findings, no need for sort.Slice
	failed := make([]api.CheckFinding, 0, r.Summary.Failed)
	for _, f := range r.Findings {
		if f.Status == "failed" {
			failed = append(failed, f)
		}
	}
	for i := 1; i < len(failed); i++ {
		j := i
		for j > 0 && severityRank(failed[j].Severity) > severityRank(failed[j-1].Severity) {
			failed[j], failed[j-1] = failed[j-1], failed[j]
			j--
		}
	}

	for _, f := range failed {
		fmt.Fprintf(out, "  [%s] %s\n", up(f.Severity), f.CheckName)
		if f.Resource != "" {
			fmt.Fprintf(out, "    Resource: %s\n", f.Resource)
		}
		if f.FilePath != "" {
			loc := f.FilePath
			if f.LineRange.Start > 0 {
				loc = fmt.Sprintf("%s:%d-%d", f.FilePath, f.LineRange.Start, f.LineRange.End)
			}
			fmt.Fprintf(out, "    At: %s\n", loc)
		}
		fmt.Fprintf(out, "    %s\n", f.CheckID)
		if f.Guideline != "" {
			fmt.Fprintf(out, "    Fix: %s\n", f.Guideline)
		}
		fmt.Fprintln(out)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func up(s string) string {
	// Inline upper-caser; avoids pulling strings.ToUpper for one call site.
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
