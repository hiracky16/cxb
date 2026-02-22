package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Checks the freshness and quality of your documentation.",
	Long: `This command lints your documentation based on rules defined in cxb.yml.

It performs the following checks:
1. Freshness: Ensures documentation is not older than the code it links to,
   and that the document itself has been updated recently.
2. Dead Links: Detects links pointing to non-existent files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		issues, err := RunAnalysis()
		if err != nil {
			// Errors during the run (e.g., cannot read config) are fatal.
			return fmt.Errorf("error during check: %w", err)
		}

		hasErrors := false
		for _, issue := range issues {
			lineInfo := ""
			if issue.Line > 0 {
				lineInfo = fmt.Sprintf(":%d", issue.Line)
			}
			cmd.Printf("[%s] %s%s: %s\n", issue.Severity, issue.File, lineInfo, issue.Message)
			if issue.Severity == "ERROR" {
				hasErrors = true
			}
		}

		if len(issues) > 0 {
			cmd.Println()
		}

		if hasErrors {
			return fmt.Errorf("found %d issues (%d errors)", len(issues), countErrors(issues))
		} else if len(issues) > 0 {
			cmd.Println("✅ Found only warnings, but no errors.")
		} else {
			cmd.Println("✅ All documents are fresh and links are valid!")
		}

		return nil
	},
}

func countErrors(issues []Issue) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == "ERROR" {
			count++
		}
	}
	return count
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
