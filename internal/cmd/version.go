package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version and build info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ops0 %s (%s, built %s)\n", buildVersion, buildCommit, buildDate)
	},
}
