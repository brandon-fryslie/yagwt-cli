package commands

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version, git commit, and build date of yagwt.`,
	Run: func(cmd *cobra.Command, args []string) {
		initFormatter()
		output := formatter.FormatVersion(Version, GitCommit, BuildDate)
		printOutput(output)
	},
}
