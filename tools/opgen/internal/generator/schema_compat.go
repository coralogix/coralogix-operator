package generator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"
)

const schemaApprovalsFile = "tools/opgen/approved-schema-changes.yaml"

type schemaApproval struct {
	Kind   string `yaml:"kind"`
	Path   string `yaml:"path"`
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Reason string `yaml:"reason"`
}

type schemaApprovals struct {
	Changes []schemaApproval `yaml:"changes"`
}

type schemaChange struct {
	Kind, Path, Before, After string
}

// CheckReleasedSchema compares rules on existing CRD paths with the PR base. New paths are free.
func CheckReleasedSchema(start, base string) error {
	root, err := operatorRoot(start)
	if err != nil {
		return err
	}
	if base == "" {
		return fmt.Errorf("the base commit is required")
	}
	verify := exec.Command("git", "rev-parse", "--verify", base+"^{commit}")
	verify.Dir = root
	if output, err := verify.CombinedOutput(); err != nil {
		return fmt.Errorf("verify base commit: %w: %s", err, strings.TrimSpace(string(output)))
	}
	kinds, err := readKinds(root)
	if err != nil {
		return err
	}
	approvals, err := readSchemaApprovals(filepath.Join(root, schemaApprovalsFile))
	if err != nil {
		return err
	}
	var changes []schemaChange
	for _, kind := range kinds {
		path := "config/crd/bases/coralogix.com_" + plural(kind.Kind) + ".yaml"
		old, err := gitFile(root, base, path)
		if err != nil {
			return err
		}
		if old == nil {
			continue // a new CRD has no released fields
		}
		current, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		before, err := crdVersions(old)
		if err != nil {
			return fmt.Errorf("base %s: %w", path, err)
		}
		after, err := crdVersions(current)
		if err != nil {
			return fmt.Errorf("current %s: %w", path, err)
		}
		for version, oldSpec := range before {
			newSpec := after[version]
			if err := compareSchema(kind.Kind, "spec.versions."+version+".schema.openAPIV3Schema.properties.spec", oldSpec, newSpec, &changes); err != nil {
				return fmt.Errorf("compare %s: %w", path, err)
			}
		}
	}
	used := map[int]bool{}
	var rejected []string
	for _, change := range changes {
		allowed := false
		for i, approval := range approvals.Changes {
			if approval.Kind == change.Kind && approval.Path == change.Path && approval.Before == change.Before && approval.After == change.After && approval.Reason != "" {
				used[i], allowed = true, true
				break
			}
		}
		if !allowed {
			rejected = append(rejected, fmt.Sprintf("%s %s: %s -> %s", change.Kind, change.Path, change.Before, change.After))
		}
	}
	for i, approval := range approvals.Changes {
		if !used[i] {
			rejected = append(rejected, fmt.Sprintf("unused schema approval for %s %s", approval.Kind, approval.Path))
		}
	}
	if len(rejected) != 0 {
		slices.Sort(rejected)
		return fmt.Errorf("released CRD rules changed; review each change and add its hashes and reason to %s:\n%s", schemaApprovalsFile, strings.Join(rejected, "\n"))
	}
	return nil
}

func operatorRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "tools/opgen/kinds.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot find the operator root from %s", start)
		}
		dir = parent
	}
}

func gitFile(root, base, path string) ([]byte, error) {
	check := exec.Command("git", "ls-tree", "--name-only", base, "--", path)
	check.Dir = root
	listed, err := check.Output()
	if err != nil {
		return nil, fmt.Errorf("list base %s: %w", path, err)
	}
	if strings.TrimSpace(string(listed)) == "" {
		return nil, nil
	}
	read := exec.Command("git", "show", base+":"+path)
	read.Dir = root
	data, err := read.Output()
	if err != nil {
		return nil, fmt.Errorf("read base %s: %w", path, err)
	}
	return data, nil
}

func readSchemaApprovals(path string) (schemaApprovals, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return schemaApprovals{}, nil
	}
	if err != nil {
		return schemaApprovals{}, err
	}
	var approvals schemaApprovals
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&approvals); err != nil {
		return schemaApprovals{}, fmt.Errorf("read schema approvals: %w", err)
	}
	return approvals, nil
}

func crdVersions(data []byte) (map[string]map[string]any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	versions, ok := nestedMap(doc, "spec")["versions"].([]any)
	if !ok {
		return nil, fmt.Errorf("CRD has no versions")
	}
	out := map[string]map[string]any{}
	for _, raw := range versions {
		version, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid CRD version")
		}
		name, ok := version["name"].(string)
		if !ok {
			return nil, fmt.Errorf("CRD version has no name")
		}
		out[name] = nestedMap(nestedMap(nestedMap(nestedMap(version, "schema"), "openAPIV3Schema"), "properties"), "spec")
		if out[name] == nil {
			return nil, fmt.Errorf("CRD version %s has no spec schema", name)
		}
	}
	return out, nil
}

func nestedMap(parent map[string]any, key string) map[string]any {
	child, _ := parent[key].(map[string]any)
	return child
}

func compareSchema(kind, path string, old, current map[string]any, changes *[]schemaChange) error {
	if old == nil && current == nil {
		return nil
	}
	before, err := ruleHash(old)
	if err != nil {
		return fmt.Errorf("%s before: %w", path, err)
	}
	after, err := ruleHash(current)
	if err != nil {
		return fmt.Errorf("%s after: %w", path, err)
	}
	if old == nil || current == nil {
		*changes = append(*changes, schemaChange{Kind: kind, Path: path, Before: before, After: after})
		return nil
	}
	if before != after {
		*changes = append(*changes, schemaChange{Kind: kind, Path: path, Before: before, After: after})
	}
	for name, oldChild := range nestedMap(old, "properties") {
		child, _ := oldChild.(map[string]any)
		currentChild, _ := nestedMap(current, "properties")[name].(map[string]any)
		if err := compareSchema(kind, path+".properties."+name, child, currentChild, changes); err != nil {
			return err
		}
	}
	for _, part := range []struct{ key, suffix string }{{"items", "[]"}, {"additionalProperties", "{}"}} {
		if oldChild, ok := old[part.key].(map[string]any); ok {
			if err := compareSchema(kind, path+part.suffix, oldChild, nestedMap(current, part.key), changes); err != nil {
				return err
			}
		}
	}
	return nil
}

func ruleHash(schema map[string]any) (string, error) {
	if schema == nil {
		return "missing", nil
	}
	rules := map[string]any{}
	for key, value := range schema {
		switch key {
		case "properties", "items", "description", "title", "example", "examples", "externalDocs":
			continue
		case "additionalProperties":
			if _, ok := value.(map[string]any); ok {
				continue
			}
		}
		rules[key] = value
	}
	data, err := json.Marshal(rules)
	if err != nil {
		return "", fmt.Errorf("encode schema rules: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8]), nil
}
