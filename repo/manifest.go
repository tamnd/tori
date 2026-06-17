package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SchemaVersion is the on-disk manifest layout version, bumped when the repo
// shape changes so a future tori can migrate an old archive (spec §6.5).
const SchemaVersion = 1

// Manifest is the repository index — the first file tori info, tori add, and
// tori render read. Its record-bearing fields are sorted deterministically so a
// re-capture of the same bytes writes a byte-identical manifest (TP5); the only
// wall-clock values live in Captures and are passed in at the surface boundary.
type Manifest struct {
	Service     string    `json:"service"`
	Target      TargetRef `json:"target"`
	TiersUsed   []string  `json:"tiers_used"`
	Tweets      int       `json:"tweets"`
	Media       int       `json:"media"`
	Threads     int       `json:"threads"`
	Range       Range     `json:"range"`
	Captures    []Capture `json:"captures"`
	MediaIndex  []Asset   `json:"media_index,omitempty"`
	ToriVersion string    `json:"tori_version"`
	Schema      int       `json:"schema"`
}

// TargetRef identifies what the repository archives (spec §6.5).
type TargetRef struct {
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	UserID string `json:"user_id,omitempty"`
	Query  string `json:"query,omitempty"`
}

// Range is the oldest/newest captured tweet timestamp.
type Range struct {
	Oldest time.Time `json:"oldest,omitempty"`
	Newest time.Time `json:"newest,omitempty"`
}

// Capture records one archive run: when it happened (the --date stamp), how many
// records it added, and which tier served them. This is the only place a
// wall-clock value lives in the manifest (TP5).
type Capture struct {
	At    string `json:"at"`
	Added int    `json:"added"`
	Tier  string `json:"tier"`
}

// Asset records where one media item went on disk, so the repository is honest
// about exactly what is and is not localised (spec §8).
type Asset struct {
	Key    string `json:"key"`
	Type   string `json:"type"`
	Path   string `json:"path,omitempty"`   // repo-relative local path, empty if not on disk
	Source string `json:"source,omitempty"` // original URL
	Status string `json:"status"`           // local | unavailable | stream-only | skipped
}

// NewManifest builds an empty manifest for a target.
func NewManifest(target TargetRef, version string) *Manifest {
	return &Manifest{
		Service:     "x",
		Target:      target,
		ToriVersion: version,
		Schema:      SchemaVersion,
	}
}

// LoadManifest reads manifest.json from a repository root, returning ok=false
// when the repo does not yet exist.
func LoadManifest(root string) (*Manifest, bool, error) {
	b, err := os.ReadFile(filepath.Join(root, ManifestFile))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false, err
	}
	return &m, true, nil
}

// Save writes the manifest deterministically: tiers and media index sorted,
// indented JSON, trailing newline.
func (m *Manifest) Save(root string) error {
	m.normalize()
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ManifestFile), b, 0o644)
}

// normalize sorts the deterministic fields and de-duplicates tiers.
func (m *Manifest) normalize() {
	m.TiersUsed = uniqSorted(m.TiersUsed)
	sort.Slice(m.MediaIndex, func(i, j int) bool {
		if m.MediaIndex[i].Key != m.MediaIndex[j].Key {
			return m.MediaIndex[i].Key < m.MediaIndex[j].Key
		}
		return m.MediaIndex[i].Source < m.MediaIndex[j].Source
	})
	if m.Schema == 0 {
		m.Schema = SchemaVersion
	}
	if m.Service == "" {
		m.Service = "x"
	}
}

// AddTier records that a tier served part of this capture.
func (m *Manifest) AddTier(tier string) {
	if tier == "" {
		return
	}
	for _, t := range m.TiersUsed {
		if t == tier {
			return
		}
	}
	m.TiersUsed = append(m.TiersUsed, tier)
}

// AddCapture appends one capture entry.
func (m *Manifest) AddCapture(at string, added int, tier string) {
	m.Captures = append(m.Captures, Capture{At: at, Added: added, Tier: tier})
	m.AddTier(tier)
}

func uniqSorted(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
