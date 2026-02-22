package cmd

import (
	"github.com/spf13/cobra"
	"os"
)

// rootCmd はベースコマンドです
var rootCmd = &cobra.Command{
	Use:   "cxb",
	Short: "cxb is a context builder for AI-native development",
	Long: `cxb helps you manage documentation freshness
and generate context for Coding Agents.`,
}

// Execute は main.go から呼ばれます
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
