package cmd

import (
	"os"
	"github.com/spf13/cobra"
)

// rootCmd はベースコマンドです
var rootCmd = &cobra.Command{
	Use:   "ctxb",
	Short: "ctxb is a context builder for AI-native development",
	Long: `ctxb helps you manage documentation freshness
and generate context for Coding Agents.`,
}

// Execute は main.go から呼ばれます
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
