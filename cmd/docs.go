package cmd

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ReportNode represents a node in the graph for the HTML report.
type ReportNode struct {
	ID         string
	Path       string
	Status     string // Healthy, Stale, Warning, Broken
	Author     string
	LastMod    string
	CommitHash string
	Tooltip    string // Detailed info for tooltip
}

// ReportEdge represents an edge in the graph for the HTML report.
type ReportEdge struct {
	From string
	To   string
}

// ReportData holds all data to be passed to the HTML template.
type ReportData struct {
	Nodes []ReportNode
	Edges []ReportEdge
}

const mermaidTemplate = `
<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<title>cxb Report</title>
<script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
<style>
.node.healthy rect { fill: #0f0; } /* Green */
.node.stale rect { fill: #f00; }    /* Red */
.node.warning rect { fill: #ff0; }  /* Yellow */
.node.broken rect { fill: #888; }   /* Gray */
</style>
</head>
<body>
<h1>cxb Dependency Graph</h1>
<pre class="mermaid">
graph TD
{{- range .Nodes }}
    {{ .ID }}["{{ .Path }}"]:::{{ .Status | lower }}
{{- end }}
{{- range .Edges }}
    {{ .From }} --> {{ .To }}
{{- end }}

{{- range .Nodes }}
    click {{ .ID }} call showNodeDetails("{{ .Path }}", "{{ .Author }}", "{{ .LastMod }}", "{{ .CommitHash }}", "{{ .Status }}", "{{ .Tooltip }}")
{{- end }}
</pre>

<div id="nodeDetails" style="position:fixed; bottom:10px; right:10px; background:white; padding:10px; border:1px solid black; display:none;">
    <h2>Node Details</h2>
    <p><strong>Path:</strong> <span id="detailPath"></span></p>
    <p><strong>Status:</strong> <span id="detailStatus"></span></p>
    <p><strong>Author:</strong> <span id="detailAuthor"></span></p>
    <p><strong>Last Modified:</strong> <span id="detailLastMod"></span></p>
    <p><strong>Commit Hash:</strong> <span id="detailCommitHash"></span></p>
    <button onclick="document.getElementById('nodeDetails').style.display='none'">Close</button>
</div>

<script>
mermaid.initialize({startOnLoad:true});

function showNodeDetails(path, author, lastMod, commitHash, status, tooltip) {
    document.getElementById('detailPath').innerText = path;
    document.getElementById('detailStatus').innerText = status;
    document.getElementById('detailAuthor').innerText = author;
    document.getElementById('detailLastMod').innerText = lastMod;
    document.getElementById('detailCommitHash').innerText = commitHash;
    document.getElementById('nodeDetails').style.display = 'block';
}
</script>
</body>
</html>
`

// docsCmd represents the docs command
var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Visualize document freshness and structure",
	Long:  `Generate an HTML report using Mermaid.js that visualizes the dependency graph and freshness of the documentation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		graph, err := BuildGraph()
		if err != nil {
			return err
		}

		// Convert Graph to ReportData
		var reportData ReportData
		nodeIDMap := make(map[string]string)
		nodeIndex := 0

		for path, node := range graph.Nodes {
			id := fmt.Sprintf("Node%d", nodeIndex)
			nodeIDMap[path] = id
			nodeIndex++

			lastModStr := "Unknown"
			if node.GitInfo.Exists {
				lastModStr = time.Unix(node.GitInfo.Timestamp, 0).Format("2006-01-02 15:04:05")
			}

			var tooltipParts []string
			for _, issue := range node.Issues {
				tooltipParts = append(tooltipParts, issue.Message)
			}
			tooltip := strings.Join(tooltipParts, " | ")
			tooltip = strings.ReplaceAll(tooltip, "\"", "'")

			reportData.Nodes = append(reportData.Nodes, ReportNode{
				ID:         id,
				Path:       node.Path,
				Status:     string(node.Status),
				Author:     node.GitInfo.Author,
				LastMod:    lastModStr,
				CommitHash: node.GitInfo.Hash,
				Tooltip:    tooltip,
			})
		}

		for _, edge := range graph.Edges {
			fromID, fromOk := nodeIDMap[edge.From]
			toID, toOk := nodeIDMap[edge.To]
			if fromOk && toOk {
				reportData.Edges = append(reportData.Edges, ReportEdge{
					From: fromID,
					To:   toID,
				})
			}
		}

		// Generate HTML
		funcMap := template.FuncMap{
			"lower": strings.ToLower,
		}
		tmpl, err := template.New("report").Funcs(funcMap).Parse(mermaidTemplate)
		if err != nil {
			return fmt.Errorf("failed to parse template: %w", err)
		}

		reportFile := "cxb-report.html"
		f, err := os.Create(reportFile)
		if err != nil {
			return fmt.Errorf("failed to create report file: %w", err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, reportData); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		cmd.Printf("✅ Generated report: %s\n", reportFile)

		// Open in browser
		_ = openBrowser(reportFile)

		return nil
	},
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
