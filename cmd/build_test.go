package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCmd(t *testing.T) {
	dir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalWd)

	// Setup cxb.yml
	os.WriteFile("cxb.yml", []byte(`
export:
  - output: ".cursorrules"
    include: ["docs/rules.md", "docs/arch/*.md"]
`), 0644)

	// Setup files
	os.MkdirAll("docs/arch", 0755)
	os.WriteFile("docs/rules.md", []byte("# Rules\nDo this."), 0644)
	os.WriteFile("docs/arch/overview.md", []byte("# Overview\nSystem architecture."), 0644)
	os.WriteFile("docs/arch/details.md", []byte("# Details\nDeep dive."), 0644)

	rootCmd.ResetFlags()
	rootCmd.SetArgs([]string{"build"})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	// Check output
	assert.FileExists(t, ".cursorrules")
	content, err := os.ReadFile(".cursorrules")
	assert.NoError(t, err)
	contentStr := string(content)

	assert.Contains(t, contentStr, "# Rules")
	assert.Contains(t, contentStr, "# Overview")
	assert.Contains(t, contentStr, "# Details")
	assert.Contains(t, contentStr, "--- File: docs/rules.md ---")
}
