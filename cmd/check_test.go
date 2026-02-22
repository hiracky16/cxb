package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupGitRepo(t *testing.T, dir string) {
	t.Helper()
	runCmd := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to run %s %v: %v\nOutput: %s", name, args, err, string(out))
		}
	}

	runCmd("git", "init")
	runCmd("git", "config", "user.email", "test@example.com")
	runCmd("git", "config", "user.name", "Test User")
}

func commitFile(t *testing.T, dir, path, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(fullPath, []byte(content), 0644)
	assert.NoError(t, err)

	runCmd := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to run %s %v: %v\nOutput: %s", name, args, err, string(out))
		}
	}

	runCmd("git", "add", path)
	runCmd("git", "commit", "-m", "commit "+path)
}

func TestCheckCmd(t *testing.T) {
	t.Run("Healthy Case", func(t *testing.T) {
		dir := t.TempDir()
		setupGitRepo(t, dir)
		
		originalWd, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(originalWd)

		// Create cxb.yml
		os.WriteFile("cxb.yml", []byte(`
paths:
  sources: ["docs"]
  targets: ["src"]
`), 0644)

		// Create and commit files in correct order
		commitFile(t, dir, "src/main.go", "package main")
		time.Sleep(1100 * time.Millisecond) // Ensure different timestamps
		commitFile(t, dir, "docs/index.md", "Link to [main](../src/main.go)")

		buf := new(bytes.Buffer)
		rootCmd.ResetFlags()
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"check"})
		
		err := rootCmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "✅ All documents are fresh and links are valid!")
	})

	t.Run("Dead Link Case", func(t *testing.T) {
		dir := t.TempDir()
		setupGitRepo(t, dir)
		
		originalWd, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(originalWd)

		os.WriteFile("cxb.yml", []byte(`
paths:
  sources: ["docs"]
  targets: ["src"]
`), 0644)

		commitFile(t, dir, "docs/index.md", "Link to [none](../src/none.go)")

		buf := new(bytes.Buffer)
		rootCmd.ResetFlags()
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"check"})
		
		err := rootCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, buf.String(), "[ERROR] docs/index.md: Dead link to '../src/none.go'")
	})

	t.Run("Stale Case (Code Newer than Doc)", func(t *testing.T) {
		dir := t.TempDir()
		setupGitRepo(t, dir)
		
		originalWd, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(originalWd)

		os.WriteFile("cxb.yml", []byte(`
paths:
  sources: ["docs"]
  targets: ["src"]
`), 0644)

		// Create and commit files: Doc first, then Code update
		commitFile(t, dir, "src/main.go", "package main")
		time.Sleep(1100 * time.Millisecond)
		commitFile(t, dir, "docs/index.md", "Link to [main](../src/main.go)")
		time.Sleep(1100 * time.Millisecond)
		commitFile(t, dir, "src/main.go", "package main // updated")

		buf := new(bytes.Buffer)
		rootCmd.ResetFlags()
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"check"})
		
		err := rootCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, buf.String(), "[ERROR] docs/index.md: Stale documentation: linked code '../src/main.go' is newer.")
	})

	t.Run("Freshness Rule Violation (Too Old)", func(t *testing.T) {
		dir := t.TempDir()
		setupGitRepo(t, dir)
		
		originalWd, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(originalWd)

		// Set error_days to 0 (or a very small number if it was possible)
		// But error_days is int. If I set it to 1, I'd need to wait a day or fake the git commit date.
		// Since faking git commit date is possible:
		
		os.WriteFile("cxb.yml", []byte(`
paths:
  sources: ["docs"]
rules:
  freshness:
    enabled: true
    error_days: 1
`), 0644)

		// commit file with old date
		fullPath := filepath.Join(dir, "docs/old.md")
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte("# Old Doc"), 0644)
		
		cmd := exec.Command("git", "add", "docs/old.md")
		cmd.Dir = dir
		cmd.Run()
		
		// 2 days ago
		oldDate := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
		cmd = exec.Command("git", "commit", "-m", "old commit", "--date", oldDate)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+oldDate)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to commit with old date: %v\nOutput: %s", err, string(out))
		}

		buf := new(bytes.Buffer)
		rootCmd.ResetFlags()
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"check"})
		
		err := rootCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, buf.String(), "[ERROR] docs/old.md: Documentation not updated in 2 days (threshold is 1).")
	})
}
