package generator

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/version"
)

// Marker lines of a block that the generator owns inside a handwritten file.
const (
	blockBegin = "# BEGIN generated kinds"
	blockEnd   = "# END generated kinds"
)

// ownedDirs are the directories that can hold generated files, relative to the operator root.
// Publish removes a generated file in them that the run does not write: its kind is gone.
var ownedDirs = []string{
	"api/coralogix/v1alpha1",
	"internal/controller/coralogix/v1alpha1",
	"internal/utils",
	"tests/e2e",
	"config/samples/v1alpha1/generated",
}

// output is everything that one run writes.
type output struct {
	// files are whole generated files by path relative to the operator root.
	files map[string][]byte
	// blocks are the new bodies of the marked blocks by path of the handwritten file.
	blocks map[string]string
}

// change is one file that publish writes or removes.
type change struct {
	path    string // absolute
	data    []byte // nil removes the file
	prior   []byte // nil when the file does not exist
	staged  string // the staged file, "" for a removal
	changed bool
}

var renameFile = os.Rename

// publish checks every target, stages every new file next to its target, and then moves the
// staged files into place. A check or stage error leaves every file as it was.
func publish(root string, out output) error {
	changes, err := plan(root, out)
	if err != nil {
		return err
	}
	if err := stage(changes); err != nil {
		return err
	}
	return commit(changes)
}

// plan lists the changes and checks that the generator owns every file that it replaces.
func plan(root string, out output) ([]*change, error) {
	var changes []*change
	for _, rel := range sortedKeys(out.files) {
		path := filepath.Join(root, rel)
		prior, err := readOwned(path)
		if err != nil {
			return nil, err
		}
		changes = append(changes, &change{path: path, data: out.files[rel], prior: prior})
	}
	for _, rel := range sortedKeys(out.blocks) {
		path := filepath.Join(root, rel)
		prior, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		data, err := replaceBlock(prior, out.blocks[rel])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		changes = append(changes, &change{path: path, data: data, prior: prior})
	}
	stale, err := staleFiles(root, out.files)
	if err != nil {
		return nil, err
	}
	changes = append(changes, stale...)
	for _, c := range changes {
		c.changed = c.data == nil || c.prior == nil || !bytes.Equal(c.data, c.prior)
	}
	return changes, nil
}

// readOwned returns the content of a generated file, or nil when it does not exist. A file
// without the generator header belongs to people: the generator never replaces it.
func readOwned(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s exists and is not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if !ownedData(data) {
		return nil, fmt.Errorf("%s exists and has no %s header; the generator does not replace a handwritten file", path, version.Tool)
	}
	return data, nil
}

func ownedData(data []byte) bool {
	line, _, _ := bytes.Cut(data, []byte("\n"))
	return version.Owned(string(line))
}

// staleFiles are the generated files in the owned directories that the run does not write.
func staleFiles(root string, files map[string][]byte) ([]*change, error) {
	var out []*change
	for _, dir := range ownedDirs {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", dir, err)
		}
		for _, e := range entries {
			rel := filepath.Join(dir, e.Name())
			if _, ok := files[rel]; ok || !e.Type().IsRegular() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", rel, err)
			}
			if ownedData(data) {
				out = append(out, &change{path: filepath.Join(root, rel), prior: data})
			}
		}
	}
	return out, nil
}

// replaceBlock replaces the lines between the marker lines with body. The file must have each
// marker exactly once, in order.
func replaceBlock(data []byte, body string) ([]byte, error) {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	begin, end := -1, -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case blockBegin:
			if begin >= 0 {
				return nil, fmt.Errorf("more than one %q line", blockBegin)
			}
			begin = i
		case blockEnd:
			if end >= 0 {
				return nil, fmt.Errorf("more than one %q line", blockEnd)
			}
			end = i
		}
	}
	if begin < 0 || end < 0 || end < begin {
		return nil, fmt.Errorf("the file needs one %q line and one %q line after it", blockBegin, blockEnd)
	}
	var b strings.Builder
	for _, line := range lines[:begin+1] {
		b.WriteString(line + "\n")
	}
	b.WriteString(body)
	for _, line := range lines[end:] {
		b.WriteString(line + "\n")
	}
	return []byte(b.String()), nil
}

// stage writes each new file next to its target. On error, it removes what it wrote.
func stage(changes []*change) error {
	for _, c := range changes {
		if !c.changed || c.data == nil {
			continue
		}
		dir := filepath.Dir(c.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			unstage(changes)
			return fmt.Errorf("create %s: %w", dir, err)
		}
		f, err := os.CreateTemp(dir, "."+filepath.Base(c.path)+".opgen-stage-")
		if err != nil {
			unstage(changes)
			return fmt.Errorf("stage %s: %w", c.path, err)
		}
		c.staged = f.Name()
		_, err = f.Write(c.data)
		err = errors.Join(err, f.Chmod(0o644), f.Close())
		if err != nil {
			unstage(changes)
			return fmt.Errorf("stage %s: %w", c.path, err)
		}
	}
	return nil
}

func unstage(changes []*change) {
	for _, c := range changes {
		if c.staged != "" {
			_ = os.Remove(c.staged)
			c.staged = ""
		}
	}
}

// commit moves the staged files into place and removes stale files. When a step fails, it writes
// back the content of every file that it already changed.
func commit(changes []*change) error {
	var done []*change
	for _, c := range changes {
		if !c.changed {
			continue
		}
		var err error
		if c.data == nil {
			err = os.Remove(c.path)
		} else {
			err = renameFile(c.staged, c.path)
		}
		if err != nil {
			unstage(changes)
			return errors.Join(fmt.Errorf("publish %s: %w", c.path, err), restore(done))
		}
		c.staged = ""
		done = append(done, c)
	}
	return nil
}

func restore(done []*change) error {
	var errs []error
	for _, c := range slices.Backward(done) {
		if c.prior == nil {
			errs = append(errs, os.Remove(c.path))
			continue
		}
		errs = append(errs, os.WriteFile(c.path, c.prior, 0o644))
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("restore the previous files: %w", err)
	}
	return nil
}
