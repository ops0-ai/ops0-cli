package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	loginAPIKey  string
	loginAPIBase string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with an ops0 API key",
	Long: `Authenticate by pasting an API key generated in your ops0 settings.

  brew.ops0.ai → Settings → API Keys → Generate

The key is stored at ~/.ops0/config.yaml (chmod 0600) and never logged.
For air-gapped or CI use, pass --api-key directly or set OPS0_API_KEY.`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "API key (skips interactive prompt)")
	loginCmd.Flags().StringVar(&loginAPIBase, "api-base", "", "Override the API base URL (e.g. https://staging.ops0.ai)")
}

func runLogin(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadUser()
	if err != nil {
		return err
	}

	if loginAPIBase != "" {
		cfg.APIBaseURL = loginAPIBase
	}

	// Resolve API key: flag > env > prompt. We intentionally don't echo it.
	key := loginAPIKey
	if key == "" {
		key = os.Getenv("OPS0_API_KEY")
	}
	if key == "" {
		// Always point users at the production dashboard URL even when their
		// CLI is configured against localhost — the API base and the dashboard
		// origin can differ in dev, and the friendly URL is the recoverable
		// one regardless of which mode you're in.
		fmt.Fprintln(cmd.OutOrStdout(), "Paste your ops0 API key (get one at https://brew.ops0.ai/settings?tab=api-keys):")
		// We use a regular bufio reader rather than golang.org/x/term so this
		// builds with no extra deps on Windows. Trade-off: key is visible
		// while typing. Most users will paste, not type, so it's acceptable.
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read key: %w", err)
		}
		key = strings.TrimSpace(line)
	}
	if key == "" {
		return fmt.Errorf("no API key provided")
	}

	cfg.APIKey = key

	// Verify before saving so we don't persist a broken key.
	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	who, err := client.Whoami()
	if err != nil {
		return fmt.Errorf("verifying key: %w", err)
	}

	if err := config.SaveUser(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Logged in as %s (%s) via key \"%s\"\n", who.UserEmail, who.Organization, who.APIKeyName)
	if len(who.Scopes) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Scopes: %s\n", strings.Join(who.Scopes, ", "))
	}
	return nil
}
