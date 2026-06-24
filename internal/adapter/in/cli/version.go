package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the KanbanAI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("KanbanAI v%s\n", version)
	},
}
