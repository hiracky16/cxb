package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestDocsCmd(t *testing.T) {
	// 各テストの前に rootCmd の状態をリセット
	rootCmd.SetArgs([]string{})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"docs"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("execute command failed: %v", err)
	}

	output := buf.String()

	expectedKeywords := []string{
		"📊 Generating dependency graph...",
		"🚀 Opening ctxb-report.html... (Mock)",
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(output, keyword) {
			t.Errorf("Expected output to contain '%s', but it was not found in '%s'", keyword, output)
		}
	}
}
