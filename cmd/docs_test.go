package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDocsCmdGeneration(t *testing.T) {
	// Setup temp directory
	dir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalWd)

	// Create dummy cxd.yml
	err := os.WriteFile("cxb.yml", []byte(`
paths:
  sources: ["."]
  targets: ["src"]
rules:
  freshness:
    enabled: true
    warn_days: 30
    error_days: 90
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create dummy markdown files with links
	os.WriteFile("index.md", []byte("[link to detail](detail.md)"), 0644)
	os.WriteFile("detail.md", []byte("# detail page"), 0644)

	reportFile := "cxb-report.html"

	b := new(bytes.Buffer)

	// Reset and set args
	rootCmd.ResetFlags()
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"docs"})
	err = rootCmd.Execute()

	// In CI, 'open' command might fail but we don't want the test to fail
	if err != nil && !strings.Contains(err.Error(), "fork/exec") && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("cmd.Execute() failed: %v", err)
	}

	// Assert file creation
	if _, err := os.Stat(reportFile); os.IsNotExist(err) {
		t.Fatalf("Expected report file %q to be created, but it was not", reportFile)
	}

	// Assert file content
	content, _ := os.ReadFile(reportFile)
	contentStr := string(content)

	if !strings.Contains(contentStr, "<h1>cxb Dependency Graph</h1>") {
		t.Error("HTML content missing expected H1 tag")
	}

	// Because of indeterminate order, Node IDs could be Node0, Node1
	// Instead of strict exact match, check if files are presented as nodes
	if !strings.Contains(contentStr, `"index.md"]`) {
		t.Error("HTML content missing index.md node")
	}
	if !strings.Contains(contentStr, `"detail.md"]`) {
		t.Error("HTML content missing detail.md node")
	}

	// Check if there is an edge
	if !strings.Contains(contentStr, "-->") {
		t.Error("HTML content missing Mermaid edge definition")
	}

	_ = os.Remove(reportFile) // Post-cleanup
}
