package cmd

import (
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate and view dependency graph",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("📊 Generating dependency graph...")
		cmd.Println("🚀 Opening ctxb-report.html... (Mock)")
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
