// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"regexp"
	"strings"
)

// ContainsGoTemplate checks if a string contains Go template syntax
// (e.g., {{ $labels.* }}, $labels.*, {{ printf ... }}, etc.)
func ContainsGoTemplate(text string) bool {
	if text == "" {
		return false
	}

	// Check for common Go template patterns
	goTemplatePatterns := []*regexp.Regexp{
		regexp.MustCompile(`\{\{\s*\$labels\.`),           // {{ $labels.
		regexp.MustCompile(`\$labels\.\w+`),               // $labels.name
		regexp.MustCompile(`\{\{\s*printf\s+`),            // {{ printf
		regexp.MustCompile(`\{\{\s*\$value\s*\}\}`),      // {{ $value }}
		regexp.MustCompile(`\$value[^a-zA-Z0-9_]`),        // $value (not followed by word char) - Go regex doesn't support lookahead
		regexp.MustCompile(`\$value$`),                    // $value at end of string
		regexp.MustCompile(`\{\{.*\$labels`),              // {{ ... $labels
		regexp.MustCompile(`\{\{.*\$value`),               // {{ ... $value
	}

	for _, pattern := range goTemplatePatterns {
		if pattern.MatchString(text) {
			return true
		}
	}

	return false
}

// ConvertPrometheusTemplateToTera converts Prometheus-style template expressions to Tera template expressions.
// Converts:
//   - {{ $labels.<name> }} → {{ alert.groups[0].keyValues.<name> }}
//   - $labels.<name> → alert.groups[0].keyValues.<name>
//   - printf functions → Tera-compatible format
//   - $value → alert.groups[0].details.metricThreshold.avgValueOverThreshold
//   - alert.value or alertDef.value → alert.groups[0].details.metricThreshold.avgValueOverThreshold
func ConvertPrometheusTemplateToTera(text string) string {
	if text == "" {
		return text
	}

	result := text

	// Replace any existing alert.value or alertDef.value with the new format
	// This handles cases where the text already contains the old format
	alertValuePattern := regexp.MustCompile(`\{\{\s*alert\.value\s*\}\}`)
	result = alertValuePattern.ReplaceAllString(result, "{{ alert.groups[0].details.metricThreshold.avgValueOverThreshold }}")
	alertValuePattern2 := regexp.MustCompile(`\{\{alert\.value\}\}`)
	result = alertValuePattern2.ReplaceAllString(result, "{{ alert.groups[0].details.metricThreshold.avgValueOverThreshold }}")
	alertDefValuePattern := regexp.MustCompile(`\{\{\s*alertDef\.value\s*\}\}`)
	result = alertDefValuePattern.ReplaceAllString(result, "{{ alert.groups[0].details.metricThreshold.avgValueOverThreshold }}")
	alertDefValuePattern2 := regexp.MustCompile(`\{\{alertDef\.value\}\}`)
	result = alertDefValuePattern2.ReplaceAllString(result, "{{ alert.groups[0].details.metricThreshold.avgValueOverThreshold }}")
	
	// Replace alert.value or alertDef.value in expressions (not in {{ }} blocks)
	alertValueInExprPattern := regexp.MustCompile(`alert\.value([^a-zA-Z0-9_]|$)`)
	result = alertValueInExprPattern.ReplaceAllString(result, "alert.groups[0].details.metricThreshold.avgValueOverThreshold$1")
	alertDefValueInExprPattern := regexp.MustCompile(`alertDef\.value([^a-zA-Z0-9_]|$)`)
	result = alertDefValueInExprPattern.ReplaceAllString(result, "alert.groups[0].details.metricThreshold.avgValueOverThreshold$1")

	// Handle printf patterns: printf "format" $value
	printfPattern := regexp.MustCompile(`\{\{\s*printf\s+"([^"]+)"\s+\$value\s*\}\}|\{\{printf\s+"([^"]+)"\s+\$value\}\}|printf\s+"([^"]+)"\s+\$value`)
	result = printfPattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract format string
		formatMatch := regexp.MustCompile(`"([^"]+)"`)
		formatStr := formatMatch.FindStringSubmatch(match)
		if len(formatStr) < 2 {
			return match
		}
		format := formatStr[1]

		// Convert to Tera format
		var teraExpr string
		if strings.Contains(format, "%.2f") || strings.Contains(format, "%.1f") {
			teraExpr = "alert.groups[0].details.metricThreshold.avgValueOverThreshold | round(method=\"ceil\", precision=2)"
		} else if strings.Contains(format, "%f") {
			teraExpr = "alert.groups[0].details.metricThreshold.avgValueOverThreshold | round(method=\"ceil\", precision=2)"
		} else if strings.Contains(format, "%d") {
			teraExpr = "alert.groups[0].details.metricThreshold.avgValueOverThreshold | round(method=\"ceil\", precision=0)"
		} else {
			teraExpr = "alert.groups[0].details.metricThreshold.avgValueOverThreshold"
		}

		if strings.Contains(match, "{{") {
			return "{{ " + teraExpr + " }}"
		}
		return "{{ " + teraExpr + " }}"
	})

	// Replace standalone {{ $value }}
	standaloneValuePattern := regexp.MustCompile(`\{\{\s*\$value\s*\}\}`)
	result = standaloneValuePattern.ReplaceAllString(result, "{{ alert.groups[0].details.metricThreshold.avgValueOverThreshold }}")

	// Replace $value in expressions (not already converted)
	// Go regex doesn't support negative lookahead, so we check for $value not followed by word chars
	valueInExprPattern := regexp.MustCompile(`\$value([^a-zA-Z0-9_]|$)`)
	result = valueInExprPattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract the character after $value (if any)
		suffix := ""
		if len(match) > 6 { // "$value" is 6 chars
			suffix = match[6:]
		}
		
		// Check if we're in a template block
		pos := strings.Index(result, match)
		if pos == -1 {
			return match
		}
		textBefore := result[:pos]
		lastOpen := strings.LastIndex(textBefore, "{{")
		if lastOpen != -1 {
			textAfter := result[pos:]
			nextClose := strings.Index(textAfter, "}}")
			if nextClose != -1 && !strings.Contains(result[max(0, pos-20):min(len(result), pos+20)], "alert.groups[0].details.metricThreshold.avgValueOverThreshold") {
				return "alert.groups[0].details.metricThreshold.avgValueOverThreshold" + suffix
			}
		}
		return match
	})

	// Handle standalone {{ $labels.<name> }} patterns
	standalonePattern := regexp.MustCompile(`\{\{\s*\$labels\.(\w+)\s*\}\}`)
	result = standalonePattern.ReplaceAllStringFunc(result, func(match string) string {
		labelMatch := regexp.MustCompile(`\$labels\.(\w+)`)
		labelName := labelMatch.FindStringSubmatch(match)
		if len(labelName) < 2 {
			return match
		}
		if strings.HasPrefix(match, "{{ ") && strings.HasSuffix(match, " }}") {
			return "{{ alert.groups[0].keyValues." + labelName[1] + " }}"
		}
		return "{{alert.groups[0].keyValues." + labelName[1] + "}}"
	})

	// Handle $labels.<name> within complex expressions
	complexPattern := regexp.MustCompile(`\$labels\.(\w+)`)
	matches := complexPattern.FindAllStringIndex(result, -1)
	// Replace from end to start to preserve positions
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		start, end := match[0], match[1]
		// Check if already converted
		context := result[max(0, start-30):min(len(result), end+30)]
		if strings.Contains(context, "alert.groups[0].keyValues") {
			continue
		}
		// Check if in a {{ }} block that needs conversion
		textBefore := result[:start]
		lastOpen := strings.LastIndex(textBefore, "{{")
		if lastOpen != -1 {
			textAfter := result[start:]
			nextClose := strings.Index(textAfter, "}}")
			if nextClose != -1 {
				block := result[lastOpen : start+nextClose+2]
				if !strings.Contains(block, "alert.groups[0].keyValues") {
					labelName := result[start+len("$labels.") : end]
					result = result[:start] + "alert.groups[0].keyValues." + labelName + result[end:]
				}
			}
		} else {
			// Not in a block, replace directly
			labelName := result[start+len("$labels.") : end]
			result = result[:start] + "alert.groups[0].keyValues." + labelName + result[end:]
		}
	}

	// Handle malformed templates like { $labels.xxx }}
	malformedPattern := regexp.MustCompile(`\{ \$labels\.(\w+)\s*\}\}`)
	result = malformedPattern.ReplaceAllString(result, "{{ alert.groups[0].keyValues.$1 }}")

	// Clean up non-Tera expressions
	nonTeraPattern := regexp.MustCompile(`\*?<<\s*(\w+)\s*>>\*?`)
	result = nonTeraPattern.ReplaceAllString(result, "[field not supported in Tera]")

	return result
}

// SanitizeDescriptionForTera sanitizes description to ensure it's Tera-compatible.
// Removes or replaces non-Tera expressions with generic placeholders.
func SanitizeDescriptionForTera(text string) string {
	if text == "" {
		return text
	}

	result := ConvertPrometheusTemplateToTera(text)

	// Remove VALUE = {{ alert.value }} or similar patterns (but keep the new format)
	// Note: We don't remove VALUE = patterns anymore as they should use the new format
	// This is kept for backward compatibility cleanup only
	valuePattern := regexp.MustCompile(`\s*VALUE\s*=\s*\{\{\s*alert\.value\s*\}\}\s*\.?\s*`)
	result = valuePattern.ReplaceAllString(result, "")
	valuePattern2 := regexp.MustCompile(`\s*VALUE\s*=\s*\{\{alert\.value\}\}\s*\.?\s*`)
	result = valuePattern2.ReplaceAllString(result, "")
	valuePattern3 := regexp.MustCompile(`\s*VALUE\s*=\s*\{\{\s*\$value\s*\}\}\s*\.?\s*`)
	result = valuePattern3.ReplaceAllString(result, "")

	// Remove lines that only contain VALUE = ...
	lines := strings.Split(result, "\n")
	var cleanedLines []string
	valueLinePattern := regexp.MustCompile(`^\s*VALUE\s*=\s*.*$`)
	for _, line := range lines {
		if !valueLinePattern.MatchString(line) {
			cleanedLines = append(cleanedLines, line)
		}
	}
	result = strings.Join(cleanedLines, "\n")

	// Note: Environment display is now handled in the controller when adding organization prefix
	// No need to add it here to avoid duplicates

	// Clean up excessive whitespace
	whitespacePattern := regexp.MustCompile(`\n\s*\n\s*\n+`)
	result = whitespacePattern.ReplaceAllString(result, "\n\n")
	spacePattern := regexp.MustCompile(`\s+`)
	result = spacePattern.ReplaceAllString(result, " ")
	result = strings.TrimSpace(result)

	// If description is empty or only contains non-Tera placeholders, provide a default
	if result == "" || result == "[field not supported in Tera]" {
		return "Alert description (field conversion not supported)"
	}

	return result
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

