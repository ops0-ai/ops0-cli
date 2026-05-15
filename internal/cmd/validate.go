package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/spf13/cobra"
)

// `ops0 validate` runs the full server-side IaC pipeline (init + validate +
// tflint) against the working directory's `.tf` / `.tofu` / `.hcl` files.
//
// This is heavier than `policies check` and is intended to run ONCE per
// Claude turn via the `Stop` hook, not per file edit.

var (
	validateFormat     string
	validateIacType    string
	validateProvider   string
	validateFailOnWarn bool
)

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Run the full IaC validate + tflint pipeline server-side",
	Long: `Bundles .tf / .tofu / .hcl files at the given path (default: cwd) and
asks the ops0 platform to run:

  1. terraform / tofu / oxid init (downloads providers in a sandbox)
  2. terraform / tofu / oxid validate
  3. tflint (provider-aware lint)

Returns unified findings. Exit code is non-zero if validate failed or any
tflint error is present. tflint warnings/notices don't block by default.

Designed to be called from Claude Code's Stop hook so end-of-turn
validation runs automatically after the agent finishes writing IaC.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVar(&validateFormat, "format", "pretty", "Output format: pretty | json")
	validateCmd.Flags().StringVar(&validateIacType, "iac-type", "terraform", "IaC flavor: terraform | opentofu | oxid")
	validateCmd.Flags().StringVar(&validateProvider, "cloud", "", "Cloud provider hint for tflint plugins: aws | gcp | azure | oracle")
	validateCmd.Flags().BoolVar(&validateFailOnWarn, "fail-on-warning", false, "Also exit non-zero on tflint warnings (default: errors only)")
}

func runValidate(cmd *cobra.Command, args []string) error {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	userCfg, err := config.LoadUser()
	if err != nil {
		return err
	}
	if userCfg.APIKey == "" {
		return fmt.Errorf("not logged in — run `ops0 login` first")
	}

	// Bundle IaC files. We reuse `collectIacFiles` from policies.go so the
	// same ignore rules and size cap apply.
	checkFiles, err := collectIacFiles(target)
	if err != nil {
		return err
	}
	if len(checkFiles) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No .tf / .tofu / .hcl files under "+target+" — nothing to validate.")
		return nil
	}

	// Convert to the map shape /validate-files expects. The check endpoint
	// uses an array because Checkov wanted positional metadata; validate
	// just needs path -> content.
	files := make(map[string]string, len(checkFiles))
	for _, f := range checkFiles {
		files[f.Name] = f.Content
	}

	client := api.New(userCfg.APIBaseURL, userCfg.APIKey)
	start := time.Now()
	result, err := client.ValidateIaC(&api.ValidateRequest{
		Files:         files,
		IacType:       validateIacType,
		CloudProvider: validateProvider,
	})
	if err != nil {
		return fmt.Errorf("validate failed: %w", err)
	}
	if result.Error != "" {
		return fmt.Errorf("validate error: %s", result.Error)
	}
	duration := time.Since(start)

	if validateFormat == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	} else {
		printValidateResult(cmd, result, target, len(checkFiles), duration)
	}

	// Telemetry — best-effort, never blocks. We only post if there's
	// something to record (validate failure or tflint findings). A clean
	// run doesn't produce a row, matching how `policies check` only ships
	// failed findings.
	if userCfg.Telemetry && shouldReportValidate(result) {
		repoCfg, repoRoot, _ := config.FindRepo(target)
		_ = repoCfg // currently only used for the hash; left for future use
		hashSrc := repoRoot
		if hashSrc == "" {
			hashSrc, _ = os.Getwd()
		}
		hash := sha256.Sum256([]byte(hashSrc))
		_ = client.ReportValidate(&api.ValidateReport{
			Validate:   result.Validate,
			Tflint:     result.Tflint,
			RepoHash:   hex.EncodeToString(hash[:]),
			CLIVersion: buildVersion,
		})
	}

	// Exit rules:
	//   - terraform validate failed     -> exit 1 (always)
	//   - tflint errors > 0             -> exit 1
	//   - tflint warnings > 0 + --fail-on-warning -> exit 1
	hardFail := !result.Validate.Valid
	if result.Tflint != nil {
		if result.Tflint.Summary.Errors > 0 {
			hardFail = true
		}
		if validateFailOnWarn && result.Tflint.Summary.Warnings > 0 {
			hardFail = true
		}
	}
	if hardFail {
		os.Exit(1)
	}
	return nil
}

// printValidateResult renders the validate + tflint sections in a compact
// human-readable form. Sent to stderr by callers so Claude Code's Stop
// hook surfaces the output to the model on a non-zero exit.
func printValidateResult(cmd *cobra.Command, r *api.ValidateResponse, target string, fileCount int, duration time.Duration) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "ops0 validate %s (%d files, %s)\n\n", target, fileCount, duration.Round(time.Millisecond))

	// terraform / tofu / oxid validate block
	if r.Validate.Valid {
		fmt.Fprintln(out, "✓ Configuration is valid")
	} else {
		fmt.Fprintln(out, "✗ Validation failed:")
		for _, e := range r.Validate.Errors {
			fmt.Fprintf(out, "  - %s\n", trimLines(e, 4))
		}
	}
	for _, w := range r.Validate.Warnings {
		fmt.Fprintf(out, "  ! %s\n", trimLines(w, 2))
	}

	// tflint block
	if r.Tflint == nil {
		fmt.Fprintln(out, "\ntflint: unavailable")
		return
	}
	t := r.Tflint
	fmt.Fprintf(out, "\ntflint: %d error(s), %d warning(s), %d notice(s)\n",
		t.Summary.Errors, t.Summary.Warnings, t.Summary.Notices)

	if len(t.Findings) == 0 {
		return
	}
	// Print up to ~20 most severe findings so the model isn't flooded.
	max := 20
	if len(t.Findings) < max {
		max = len(t.Findings)
	}
	for i := 0; i < max; i++ {
		f := t.Findings[i]
		loc := f.FilePath
		if f.LineRange.Start > 0 {
			loc = fmt.Sprintf("%s:%d", f.FilePath, f.LineRange.Start)
		}
		fmt.Fprintf(out, "  [%s] %s — %s (%s)\n", strings.ToUpper(f.Severity), f.RuleName, f.Message, loc)
	}
	if len(t.Findings) > max {
		fmt.Fprintf(out, "  ...and %d more (use --format=json to see all)\n", len(t.Findings)-max)
	}
}

// shouldReportValidate returns true if the response contains anything worth
// putting in the audit trail: a failed validate, or any tflint finding.
// A clean pipeline produces no row.
func shouldReportValidate(r *api.ValidateResponse) bool {
	if r == nil {
		return false
	}
	if !r.Validate.Valid {
		return true
	}
	if r.Tflint != nil && len(r.Tflint.Findings) > 0 {
		return true
	}
	return false
}

func trimLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + " ..."
}

// Used to deduplicate filepath import lint when no other consumer needs it.
var _ = filepath.Separator
