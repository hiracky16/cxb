package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const ctxbConfigContent = `name: 'my_project'
version: '0.1.0'

paths:
  sources: ["docs"]    # ドキュメントのルート
  targets: ["src"]     # 監視対象コードのルート

rules:
  freshness:
    enabled: true
    warn_days: 30      # コード更新がなくても、30日経過で警告
    error_days: 90     # 90日でエラー

export:
  - output: ".cursorrules"
    include: ["docs/rules.md", "docs/arch/**"]
`

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new ctxb project.",
	Long: `Initialize a new ctxb project.
This command creates a ctxb.yml file in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := "ctxb.yml"
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("'%s' already exists in the current directory", filePath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check for %s: %w", filePath, err)
		}

		err := os.WriteFile(filePath, []byte(ctxbConfigContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write to %s: %w", filePath, err)
		}

		        // Use cmd.OutOrStdout() to print the message, so it can be captured in tests.
		        cmd.OutOrStdout().Write([]byte(fmt.Sprintf("OK: Created configuration file: %s\n", filePath)))
		        return nil	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
