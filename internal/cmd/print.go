package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ops0-ai/ops0-cli/internal/opa"
	"github.com/spf13/cobra"
)

// printPretty renders a check result for humans. Keeps zero ANSI dependencies
// so it's safe in CI and Windows terminals; coloring can come later behind
// a flag if anyone asks.
func printPretty(cmd *cobra.Command, r *opa.Result) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Target: %s\n", r.Target)
	fmt.Fprintf(out, "Policies: %d evaluated  •  %d passed  •  %d failed\n\n", r.Total, r.Passed, r.Failed)
	if len(r.Violations) == 0 {
		fmt.Fprintln(out, "✓ No violations.")
		return
	}
	for _, v := range r.Violations {
		fmt.Fprintf(out, "  [%s] %s\n", up(v.Severity), v.PolicyName)
		fmt.Fprintf(out, "    %s\n", v.Message)
		if v.Resource != "" {
			fmt.Fprintf(out, "    Resource: %s\n", v.Resource)
		}
	}
}

func printJSON(cmd *cobra.Command, r *opa.Result) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func up(s string) string {
	// Tiny inline upper-caser so we avoid pulling in strings just for one call.
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
