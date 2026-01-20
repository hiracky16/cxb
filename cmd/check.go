package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// Config defines the structure for the ctxb.yml configuration file.
type Config struct {
	Paths struct {
		Sources []string `yaml:"sources"`
		Targets []string `yaml:"targets"`
	} `yaml:"paths"`
	Rules struct {
		Freshness struct {
			Enabled   bool `yaml:"enabled"`
			WarnDays  int  `yaml:"warn_days"`
			ErrorDays int  `yaml:"error_days"`
		} `yaml:"freshness"`
	} `yaml:"rules"`
}

// Issue represents a single problem found by the linter.
type Issue struct {
	File     string
	Line     int
	Severity string // "ERROR" or "WARNING"
	Message  string
}

// LinkInfo represents an extracted link from a markdown file.
type LinkInfo struct {
	Destination string
	Line        int
}

// GitInfo holds the last commit timestamp for a file.
type GitInfo struct {
	Timestamp int64
	Exists    bool // False if the file is not tracked by Git.
}

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Checks the freshness and quality of your documentation.",
	Long: `This command lints your documentation based on rules defined in ctxb.yml.

It performs the following checks:
1. Freshness: Ensures documentation is not older than the code it links to,
   and that the document itself has been updated recently.
2. Dead Links: Detects links pointing to non-existent files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		issues, err := runCheck()
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
			fmt.Printf("[%s] %s%s: %s
", issue.Severity, issue.File, lineInfo, issue.Message)
			if issue.Severity == "ERROR" {
				hasErrors = true
			}
		}

		if len(issues) > 0 {
			fmt.Println()
		}

		if hasErrors {
			return fmt.Errorf("found %d issues (%d errors)", len(issues), countErrors(issues))
		} else if len(issues) > 0 {
			fmt.Println("✅ Found only warnings, but no errors.")
		} else {
			fmt.Println("✅ All documents are fresh and links are valid!")
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

func runCheck() ([]Issue, error) {
	cfg, err := loadConfig("ctxb.yml")
	if err != nil {
		return nil, err
	}

	markdownFiles, err := findMarkdownFiles(cfg.Paths.Sources)
	if err != nil {
		return nil, err
	}

	var allIssues []Issue
	for _, mdFile := range markdownFiles {
		issues, err := analyzeFile(mdFile, cfg)
		if err != nil {
			allIssues = append(allIssues, Issue{File: mdFile, Severity: "ERROR", Message: fmt.Sprintf("Cannot analyze file: %v", err)})
			continue
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

func analyzeFile(filePath string, cfg *Config) ([]Issue, error) {
	var issues []Issue

	docGitInfo, err := getFileGitInfo(filePath)
	if err != nil {
		issues = append(issues, Issue{File: filePath, Severity: "WARNING", Message: fmt.Sprintf("Could not get Git history: %v. Freshness checks will be skipped.", err)})
		docGitInfo = GitInfo{Exists: false}
	}

	links, err := extractLinks(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not extract links: %w", err)
	}

	for _, link := range links {
		targetPath := filepath.Join(filepath.Dir(filePath), link.Destination)
		targetPath = filepath.Clean(targetPath)

		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			issues = append(issues, Issue{
				File:     filePath,
				Line:     link.Line,
				Severity: "ERROR",
				Message:  fmt.Sprintf("Dead link to '%s'", link.Destination),
			})
			continue
		}

		if docGitInfo.Exists && isTarget(targetPath, cfg.Paths.Targets) {
			codeGitInfo, err := getFileGitInfo(targetPath)
			if err != nil {
				issues = append(issues, Issue{File: filePath, Line: link.Line, Severity: "WARNING", Message:  fmt.Sprintf("Could not get Git history for linked file '%s': %v", link.Destination, err)})
				continue
			}
			if !codeGitInfo.Exists {
				issues = append(issues, Issue{File: filePath, Line: link.Line, Severity: "WARNING", Message: fmt.Sprintf("Linked file '%s' is not tracked by Git.", link.Destination)})
				continue
			}

			if docGitInfo.Timestamp < codeGitInfo.Timestamp {
				issues = append(issues, Issue{
					File:     filePath,
					Line:     link.Line,
					Severity: "ERROR",
					Message:  fmt.Sprintf("Stale documentation: linked code '%s' is newer.", link.Destination),
				})
			}
		}
	}

	if cfg.Rules.Freshness.Enabled && docGitInfo.Exists {
		now := time.Now().Unix()
		daysSinceUpdate := (now - docGitInfo.Timestamp) / (60 * 60 * 24)

		if cfg.Rules.Freshness.ErrorDays > 0 && daysSinceUpdate >= int64(cfg.Rules.Freshness.ErrorDays) {
			issues = append(issues, Issue{File: filePath, Severity: "ERROR", Message: fmt.Sprintf("Documentation not updated in %d days (threshold is %d).", daysSinceUpdate, cfg.Rules.Freshness.ErrorDays)})
		} else if cfg.Rules.Freshness.WarnDays > 0 && daysSinceUpdate >= int64(cfg.Rules.Freshness.WarnDays) {
			issues = append(issues, Issue{File: filePath, Severity: "WARNING", Message: fmt.Sprintf("Documentation not updated in %d days (threshold is %d).", daysSinceUpdate, cfg.Rules.Freshness.WarnDays)})
		}
	}

	return issues, nil
}

func isTarget(path string, targetRoots []string) bool {
	for _, root := range targetRoots {
		if strings.HasPrefix(path, root) {
			return true
		}
	}
	return false
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", path, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse YAML in %s: %w", path, err)
	}
	return &config, nil
}

func findMarkdownFiles(sourceDirs []string) ([]string, error) {
	var markdownFiles []string
	for _, dir := range sourceDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil { return err }
			if !d.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".markdown")) {
				markdownFiles = append(markdownFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", dir, err)
		}
	}
	return markdownFiles, nil
}

func extractLinks(filePath string) ([]LinkInfo, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	parser := goldmark.DefaultParser()
	doc := parser.Parse(text.NewReader(source))

	var links []LinkInfo
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if link, ok := n.(*ast.Link); ok {
				dest := string(link.Destination)
				if !strings.HasPrefix(dest, "http://") && !strings.HasPrefix(dest, "https://") && !strings.HasPrefix(dest, "#") {
					links = append(links, LinkInfo{Destination: dest, Line: n.Lines().At(0).Start})
				}
			}
		}
		return ast.WalkContinue, nil
	})

	return links, nil
}

func getFileGitInfo(filePath string) (GitInfo, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct", "--", filePath)
	output, err := cmd.Output()
	if err != nil { return GitInfo{Exists: false}, nil }

	timestampStr := strings.TrimSpace(string(output))
	if timestampStr == "" { return GitInfo{Exists: false}, nil }

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil { return GitInfo{}, fmt.Errorf("could not parse git timestamp '%s': %w", timestampStr, err) }

	return GitInfo{Timestamp: timestamp, Exists: true}, nil
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
