package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"
)

// toolDir is the directory of the generator in the operator repository.
const toolDir = "tools/opgen"

// kindsFile lists the kinds that the generator writes, relative to the operator root.
const kindsFile = toolDir + "/kinds.yaml"

// kindsDir holds one directory per kind that has handwritten generator input, relative to the
// operator root.
const kindsDir = toolDir + "/kinds"

// kindEntry is one generated kind.
type kindEntry struct {
	// Kind is the custom resource kind, for example ViewFolder.
	Kind string `yaml:"kind"`
	// Resource is the OpenAPI component of the API resource, for example ViewFolder.
	Resource string `yaml:"resource"`
}

type kindList struct {
	Kinds []kindEntry `yaml:"kinds"`
}

// readKinds reads the kinds file. The list is strict: an unknown key, an invalid or repeated kind,
// and a directory under kinds/ that names no listed kind are errors.
func readKinds(root string) ([]kindEntry, error) {
	data, err := os.ReadFile(filepath.Join(root, kindsFile))
	if err != nil {
		return nil, fmt.Errorf("read the list of generated kinds: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var list kindList
	if err := dec.Decode(&list); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %w", kindsFile, err)
	}
	seen := map[string]bool{}
	for _, k := range list.Kinds {
		if !token.IsIdentifier(k.Kind) || !token.IsExported(k.Kind) {
			return nil, fmt.Errorf("%s: kind %q is not an exported Go name", kindsFile, k.Kind)
		}
		if k.Resource == "" {
			return nil, fmt.Errorf("%s: kind %s has no resource", kindsFile, k.Kind)
		}
		if seen[strings.ToLower(k.Kind)] {
			return nil, fmt.Errorf("%s: kind %s is listed twice", kindsFile, k.Kind)
		}
		seen[strings.ToLower(k.Kind)] = true
	}
	entries, err := os.ReadDir(filepath.Join(root, kindsDir))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", kindsDir, err)
	}
	for _, e := range entries {
		if !e.IsDir() || !seen[e.Name()] {
			return nil, fmt.Errorf("%s/%s belongs to no kind of %s", kindsDir, e.Name(), kindsFile)
		}
	}
	kinds := slices.Clone(list.Kinds)
	slices.SortFunc(kinds, func(a, b kindEntry) int { return strings.Compare(a.Kind, b.Kind) })
	return kinds, nil
}
