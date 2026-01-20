package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckCmd(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"check"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("execute command failed: %v", err)
	}

	output := buf.String()

	expectedKeywords := []string{
		"🔍 Checking documentation freshness...",
		"✅ All docs are fresh! (Mock)",
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(output, keyword) {
			t.Errorf("Expected output to contain '%s', but it was not found in '%s'", keyword, output)
		}
	}
}
