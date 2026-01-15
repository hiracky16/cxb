package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// checkCmd の定義
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check documentation freshness and links",
	Run: func(cmd *cobra.Command, args []string) {
		// ここにロジックを実装していきます
		fmt.Println("🔍 Checking documentation freshness...")
		fmt.Println("✅ All docs are fresh! (Mock)")
	},
}

func init() {
	// rootコマンドに check を登録
	rootCmd.AddCommand(checkCmd)
}
