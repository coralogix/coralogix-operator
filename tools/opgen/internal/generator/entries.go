package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/issue"
)

// duplicateEntries reports each kind that a handwritten part of a marked file already lists. The
// generator writes the entry of every kind inside the marked block and never removes a handwritten
// one, so a kind that moved from handwritten code to generated code would be listed twice. A
// duplicate CRD entry stops `kustomize build config/default`, so the run stops before it writes.
func duplicateEntries(root string, out output) issue.Report {
	var report issue.Report
	for _, rel := range sortedKeys(out.entries) {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue // publish reports a file that it cannot read
		}
		outside := map[string]bool{}
		for _, line := range linesOutsideBlock(string(data)) {
			outside[strings.TrimSpace(line)] = true
		}
		for _, entry := range out.entries[rel] {
			if outside[entry] {
				report = append(report, issue.Issue{Code: codeDuplicateEntry, Location: rel,
					Message:     fmt.Sprintf("the handwritten part of the file already has the line %q of a generated kind", entry),
					Remediation: "Remove the handwritten entry of the generated kind from " + rel + "."})
			}
		}
	}
	return report
}

// linesOutsideBlock returns the lines before the begin marker and after the end marker.
func linesOutsideBlock(text string) []string {
	var out []string
	inside := false
	for _, line := range strings.Split(text, "\n") {
		switch strings.TrimSpace(line) {
		case blockBegin:
			inside = true
		case blockEnd:
			inside = false
		default:
			if !inside {
				out = append(out, line)
			}
		}
	}
	return out
}
