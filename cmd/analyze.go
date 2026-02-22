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

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// Config defines the structure for the cxb.yml configuration file.
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
	Export []ExportConfig `yaml:"export"`
}

// ExportConfig defines how to export context for AI agents.
type ExportConfig struct {
	Output  string   `yaml:"output"`
	Include []string `yaml:"include"`
}

// Issue represents a single problem found.
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

// GitInfo holds the last commit information for a file.
type GitInfo struct {
	Timestamp int64
	Author    string
	Hash      string
	Exists    bool // False if the file is not tracked by Git.
}

// NodeStatus represents the health status of a file.
type NodeStatus string

const (
	Healthy NodeStatus = "Healthy"
	Stale   NodeStatus = "Stale"
	Warning NodeStatus = "Warning"
	Broken  NodeStatus = "Broken"
)

// AnalysisNode represents a file in the project.
type AnalysisNode struct {
	Path    string
	GitInfo GitInfo
	Status  NodeStatus
	Issues  []Issue // Issues specific to this file
}

// AnalysisEdge represents a link between two files.
type AnalysisEdge struct {
	From string // Path of the source file
	To   string // Path of the target file
}

// AnalysisGraph represents the entire dependency graph of the project.
type AnalysisGraph struct {
	Nodes map[string]*AnalysisNode // Map from file path to node
	Edges []AnalysisEdge
}

// NewAnalysisGraph creates an empty analysis graph.
func NewAnalysisGraph() *AnalysisGraph {
	return &AnalysisGraph{
		Nodes: make(map[string]*AnalysisNode),
		Edges: []AnalysisEdge{},
	}
}

// GetOrCreateNode retrieves a node for a given path, creating it if it doesn't exist.
func (g *AnalysisGraph) GetOrCreateNode(path string) *AnalysisNode {
	if _, ok := g.Nodes[path]; !ok {
		gitInfo, _ := GetFileGitInfo(path)
		g.Nodes[path] = &AnalysisNode{
			Path:    path,
			GitInfo: gitInfo,
			Status:  Healthy, // Default status
			Issues:  []Issue{},
		}
	}
	return g.Nodes[path]
}

// RunAnalysis performs the full analysis and returns a flat list of issues.
// It's a wrapper around BuildGraph for the 'check' command.
func RunAnalysis() ([]Issue, error) {
	graph, err := BuildGraph()
	if err != nil {
		return nil, err
	}

	var allIssues []Issue
	for _, node := range graph.Nodes {
		allIssues = append(allIssues, node.Issues...)
	}
	return allIssues, nil
}

// BuildGraph constructs the full dependency graph of the project.
func BuildGraph() (*AnalysisGraph, error) {
	cfg, err := loadConfig("cxb.yml")
	if err != nil {
		return nil, err
	}

	graph := NewAnalysisGraph()

	markdownFiles, err := findMarkdownFiles(cfg.Paths.Sources)
	if err != nil {
		return nil, err
	}

	// First pass: create all nodes from markdown files
	for _, mdFile := range markdownFiles {
		graph.GetOrCreateNode(mdFile)
	}

	// Second pass: analyze files, create edges for links, and update status
	for _, mdFile := range markdownFiles {
		AnalyzeFileForGraph(mdFile, cfg, graph)
	}

	return graph, nil
}

// AnalyzeFileForGraph analyzes a single markdown file and populates the graph.
func AnalyzeFileForGraph(filePath string, cfg *Config, graph *AnalysisGraph) {
	docNode := graph.GetOrCreateNode(filePath)

	if !docNode.GitInfo.Exists {
		issue := Issue{File: filePath, Severity: "WARNING", Message: "Could not get Git history. Freshness checks will be skipped."}
		docNode.Issues = append(docNode.Issues, issue)
		docNode.Status = Warning
	}

	links, err := extractLinks(filePath)
	if err != nil {
		issue := Issue{File: filePath, Severity: "ERROR", Message: fmt.Sprintf("Cannot analyze file: %v", err)}
		docNode.Issues = append(docNode.Issues, issue)
		docNode.Status = Broken
		return
	}

	hasError := false
	hasWarning := false

	for _, link := range links {
		targetPath := filepath.Join(filepath.Dir(filePath), link.Destination)
		targetPath = filepath.Clean(targetPath)

		// This creates the node for the linked file (even if it's not a markdown file)
		targetNode := graph.GetOrCreateNode(targetPath)
		graph.Edges = append(graph.Edges, AnalysisEdge{From: filePath, To: targetPath})

		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			issue := Issue{File: filePath, Line: link.Line, Severity: "ERROR", Message: fmt.Sprintf("Dead link to '%s'", link.Destination)}
			docNode.Issues = append(docNode.Issues, issue)
			hasError = true
			continue
		}

		if docNode.GitInfo.Exists && isTarget(targetPath, cfg.Paths.Targets) {
			if !targetNode.GitInfo.Exists {
				issue := Issue{File: filePath, Line: link.Line, Severity: "WARNING", Message: fmt.Sprintf("Linked file '%s' is not tracked by Git.", link.Destination)}
				docNode.Issues = append(docNode.Issues, issue)
				hasWarning = true
				continue
			}

			if docNode.GitInfo.Timestamp < targetNode.GitInfo.Timestamp {
				issue := Issue{File: filePath, Line: link.Line, Severity: "ERROR", Message: fmt.Sprintf("Stale documentation: linked code '%s' is newer.", link.Destination)}
				docNode.Issues = append(docNode.Issues, issue)
				hasError = true
			}
		}
	}

	if cfg.Rules.Freshness.Enabled && docNode.GitInfo.Exists {
		now := time.Now().Unix()
		daysSinceUpdate := (now - docNode.GitInfo.Timestamp) / (60 * 60 * 24)

		if cfg.Rules.Freshness.ErrorDays > 0 && daysSinceUpdate >= int64(cfg.Rules.Freshness.ErrorDays) {
			issue := Issue{File: filePath, Severity: "ERROR", Message: fmt.Sprintf("Documentation not updated in %d days (threshold is %d).", daysSinceUpdate, cfg.Rules.Freshness.ErrorDays)}
			docNode.Issues = append(docNode.Issues, issue)
			hasError = true
		} else if cfg.Rules.Freshness.WarnDays > 0 && daysSinceUpdate >= int64(cfg.Rules.Freshness.WarnDays) {
			issue := Issue{File: filePath, Severity: "WARNING", Message: fmt.Sprintf("Documentation not updated in %d days (threshold is %d).", daysSinceUpdate, cfg.Rules.Freshness.WarnDays)}
			docNode.Issues = append(docNode.Issues, issue)
			hasWarning = true
		}
	}

	// Set final status based on findings
	if hasError {
		docNode.Status = Stale // Or Broken, Stale is better for "code is newer"
	} else if hasWarning {
		docNode.Status = Warning
	}
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
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cxb.yml not found. Please run 'cxb init'")
		}
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
	if len(sourceDirs) == 0 {
		sourceDirs = []string{"."}
	}
	var markdownFiles []string
	for _, dir := range sourceDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == ".git" {
				return filepath.SkipDir
			}
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
	var nodeStack []ast.Node

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			nodeStack = append(nodeStack, n) // Push
			if link, ok := n.(*ast.Link); ok {
				dest := string(link.Destination)
				if !strings.HasPrefix(dest, "http://") && !strings.HasPrefix(dest, "https://") && !strings.HasPrefix(dest, "#") {
					var line int
					// Find the parent block node to get the line number.
					for i := len(nodeStack) - 1; i >= 0; i-- {
						parent := nodeStack[i]
						if p, ok := parent.(*ast.TextBlock); ok {
							line = p.Lines().At(0).Start
							break
						}
					}
					links = append(links, LinkInfo{Destination: dest, Line: line})
				}
			}
		} else {
			nodeStack = nodeStack[:len(nodeStack)-1] // Pop
		}
		return ast.WalkContinue, nil
	})

	return links, nil
}

// GetFileGitInfo retrieves the last commit information for a given file.
func GetFileGitInfo(filePath string) (GitInfo, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct|%an|%h", "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return GitInfo{Exists: false}, nil
	}

	trimmedOutput := strings.TrimSpace(string(output))
	if trimmedOutput == "" {
		return GitInfo{Exists: false}, nil
	}

	parts := strings.Split(trimmedOutput, "|")
	if len(parts) != 3 {
		return GitInfo{}, fmt.Errorf("could not parse git log output: expected 3 parts, got %d for file %s", len(parts), filePath)
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return GitInfo{}, fmt.Errorf("could not parse git timestamp '%s': %w", parts[0], err)
	}

	return GitInfo{
		Timestamp: timestamp,
		Author:    parts[1],
		Hash:      parts[2],
		Exists:    true,
	}, nil
}
