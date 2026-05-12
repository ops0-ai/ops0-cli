package cmd

import (
	"github.com/ops0-ai/ops0-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the MCP server so agents can query ops0 policies",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server over stdio (JSON-RPC)",
	Long: `Starts an MCP server that exposes ops0 tools to compatible agents.

Wire this into Claude Code (or any MCP client) with something like:

  # ~/.config/claude/mcp.json
  {
    "mcpServers": {
      "ops0": {
        "command": "ops0",
        "args": ["mcp", "serve"]
      }
    }
  }

Tools exposed:
  list_policies        Returns the policies in scope for the current repo
  check_compliance     Evaluates Rego policies against a Terraform plan JSON
  get_finding_details  Looks up a discovery security check by templateId`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return mcp.Serve(cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
}
