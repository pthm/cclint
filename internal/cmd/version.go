package cmd

import (
	"fmt"

	"github.com/pthm/cclint/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Info())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
