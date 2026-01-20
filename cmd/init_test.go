package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// executeCommand is a helper function to execute a cobra command and capture its output.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestInitCmd(t *testing.T) {
	// Backup original commands and restore after
	originalCommands := rootCmd.Commands()
	defer func() {
		rootCmd.ResetCommands()
		rootCmd.AddCommand(originalCommands...)
	}()

	t.Run("creates ctxb.yml if it does not exist", func(t *testing.T) {
		// Reset commands for this specific test run
		rootCmd.ResetCommands()
		rootCmd.AddCommand(initCmd)

		dir := t.TempDir()
		originalWd, err := os.Getwd()
		assert.NoError(t, err)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(originalWd)

		output, err := executeCommand(rootCmd, "init")

		assert.NoError(t, err)
		assert.Equal(t, "OK: Created configuration file: ctxb.yml\n", output)

		content, err := os.ReadFile("ctxb.yml")
		assert.NoError(t, err)
		assert.Equal(t, ctxbConfigContent, string(content))
	})

	t.Run("returns an error if ctxb.yml already exists", func(t *testing.T) {
		// Reset commands for this specific test run
		rootCmd.ResetCommands()
		rootCmd.AddCommand(initCmd)

		dir := t.TempDir()

		originalWd, err := os.Getwd()
		assert.NoError(t, err)
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(originalWd)

		err = os.WriteFile("ctxb.yml", []byte("dummy content"), 0644)
		assert.NoError(t, err)

		_, err = executeCommand(rootCmd, "init")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'ctxb.yml' already exists in the current directory")
	})
}