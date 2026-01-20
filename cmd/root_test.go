package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_NoArgs(t *testing.T) {
	// rootCmd の状態をリセット
	rootCmd.SetArgs([]string{})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("execute command failed: %v", err)
	}

	output := buf.String()

	// ヘルプメッセージ（Long description）が含まれているか確認
	if !strings.Contains(output, rootCmd.Long) {
		t.Errorf("Expected output to contain long description, but it was not found.")
	}

	// Usage情報が含まれているか確認
	if !strings.Contains(output, "Usage:") {
		t.Errorf("Expected output to contain 'Usage:', but it was not found.")
	}

	// 利用可能なコマンド（checkとdocs）が表示されているか確認
	if !strings.Contains(output, "check") || !strings.Contains(output, "docs") {
		t.Errorf("Expected output to contain available commands 'check' and 'docs'.")
	}
}
