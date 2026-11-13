package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/version"
)

// shardsFile lists the shards of the live coverage suite. The workflow reads it, so a new step
// needs no workflow edit.
const shardsFile = e2eDir + "/coverage-shards.yaml"

const (
	// minutesPerStep is the estimate of the run time of one coverage step. It is a guess until the
	// first live run shows the real time.
	minutesPerStep = 1
	// shardMinutes is the time that a shard may take in tests. Setup takes about 1.5 minutes more.
	shardMinutes = 4
	// restShard runs every coverage spec that no label names, such as a handwritten one.
	restShard = "coverage-rest"
)

type shard struct {
	Name   string `json:"name"`
	Filter string `json:"filter"`
	Steps  int    `json:"steps"`
}

// shardUnit is what one shard takes whole: all steps of a Serial kind, or one step of another kind.
type shardUnit struct {
	labels []string
}

// coverageLabel is the Ginkgo label of one step of a kind.
func coverageLabel(d *kindData, step int) string {
	return fmt.Sprintf("coverage-%s-%d", d.Lower, step)
}

// packShards puts the steps of the kinds into shards of at most shardMinutes. A Serial kind stays
// in one shard, because Ginkgo Serial holds only inside one run. A kind with more steps than a shard
// takes gets a shard of its own. The last shard runs every spec that no label names.
func packShards(kinds []*kindData) []shard {
	var units []shardUnit
	var all []string
	for _, d := range kinds {
		if d.E2E == nil || len(d.E2E.Steps) == 0 {
			continue
		}
		var labels []string
		for i := range d.E2E.Steps {
			labels = append(labels, coverageLabel(d, i+1))
		}
		all = append(all, labels...)
		if d.E2E.Serial {
			units = append(units, shardUnit{labels: labels})
			continue
		}
		for _, label := range labels {
			units = append(units, shardUnit{labels: []string{label}})
		}
	}
	slices.SortStableFunc(units, func(a, b shardUnit) int {
		if len(a.labels) != len(b.labels) {
			return len(b.labels) - len(a.labels)
		}
		return strings.Compare(a.labels[0], b.labels[0])
	})
	var packed [][]string
	for _, unit := range units {
		placed := false
		for i := range packed {
			if (len(packed[i])+len(unit.labels))*minutesPerStep <= shardMinutes {
				packed[i] = append(packed[i], unit.labels...)
				placed = true
				break
			}
		}
		if !placed {
			packed = append(packed, slices.Clone(unit.labels))
		}
	}
	var out []shard
	for i, labels := range packed {
		slices.Sort(labels)
		out = append(out, shard{Name: fmt.Sprintf("coverage-%d", i+1), Filter: "coverage && (" + strings.Join(labels, " || ") + ")", Steps: len(labels)})
	}
	rest := "coverage"
	if len(all) != 0 {
		slices.Sort(all)
		rest = "coverage && !(" + strings.Join(all, " || ") + ")"
	}
	return append(out, shard{Name: restShard, Filter: rest})
}

// renderShards writes the shard file: a generator header, then JSON, which is also YAML.
func renderShards(kinds []*kindData) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(version.YAMLHeaderLine + "\n# The shards of the live coverage suite. Each filter is a Ginkgo label filter.\n")
	doc := struct {
		MinutesPerStep int     `json:"minutesPerStep"`
		Shards         []shard `json:"shards"`
	}{minutesPerStep, packShards(kinds)}
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
