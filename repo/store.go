package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamnd/x-cli/x"
)

// Store is a handle on one repository directory. It writes and reads the
// canonical records, the raw payloads, and the media files; it owns no network
// and no policy beyond the layout rules in layout.go.
type Store struct {
	Root string
}

// Open returns a Store rooted at dir, creating the directory if needed.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{Root: dir}, nil
}

// abs joins a repo-relative path onto the root, translating forward slashes to
// the OS separator.
func (s *Store) abs(rel string) string {
	return filepath.Join(s.Root, filepath.FromSlash(rel))
}

// writeFile creates parent directories and writes bytes atomically enough for a
// capture (write + rename within the same dir).
func (s *Store) writeFile(rel string, b []byte) error {
	p := s.abs(rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Exists reports whether a repo-relative file is already on disk.
func (s *Store) Exists(rel string) bool {
	_, err := os.Stat(s.abs(rel))
	return err == nil
}

// WriteTweet persists the canonical record and, when present, the untouched
// upstream payload beside it (TP3). The canonical JSON is indented and sorted by
// field for a stable, diff-friendly file.
func (s *Store) WriteTweet(t *x.Tweet, raw json.RawMessage) error {
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := s.writeFile(TweetJSON(t.ID), b); err != nil {
		return err
	}
	if len(raw) > 0 {
		if err := s.writeFile(TweetRaw(t.ID), normalizeRaw(raw)); err != nil {
			return err
		}
	}
	return nil
}

// WriteProfile persists the captured User as profile.json.
func (s *Store) WriteProfile(u *x.User) error {
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return s.writeFile(ProfileFile, b)
}

// LoadProfile reads profile.json, returning ok=false when absent.
func (s *Store) LoadProfile() (*x.User, bool, error) {
	b, err := os.ReadFile(s.abs(ProfileFile))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var u x.User
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, false, err
	}
	return &u, true, nil
}

// LoadTweets reads every canonical record under tweets/, sorted by id ascending
// so callers (render, merge, manifest) get a deterministic order (TP5).
func (s *Store) LoadTweets() ([]*x.Tweet, error) {
	dir := s.abs(TweetsDir)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".raw.json") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".json"))
	}
	sortIDs(ids)
	out := make([]*x.Tweet, 0, len(ids))
	for _, id := range ids {
		t, err := s.LoadTweet(id)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// LoadTweet reads one canonical record by id.
func (s *Store) LoadTweet(id string) (*x.Tweet, error) {
	b, err := os.ReadFile(s.abs(TweetJSON(id)))
	if err != nil {
		return nil, err
	}
	var t x.Tweet
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// HasTweet reports whether a canonical record already exists.
func (s *Store) HasTweet(id string) bool { return s.Exists(TweetJSON(id)) }

// WriteMedia writes localised media bytes to a repo-relative path.
func (s *Store) WriteMedia(rel string, b []byte) error { return s.writeFile(rel, b) }

// WriteText writes an arbitrary repo-relative text file (a rendered page, the
// CSS, an index).
func (s *Store) WriteText(rel, body string) error { return s.writeFile(rel, []byte(body)) }

// normalizeRaw re-indents a raw payload when it is valid JSON, so the sidecar is
// readable; if it is not JSON (it always should be) it is stored verbatim.
func normalizeRaw(raw json.RawMessage) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return raw
	}
	return append(b, '\n')
}

// sortIDs orders snowflake-string ids numerically (shorter is smaller, then
// lexical), so 99 sorts before 100 and the order matches chronological.
func sortIDs(ids []string) {
	sort.Slice(ids, func(i, j int) bool { return lessID(ids[i], ids[j]) })
}

// lessID compares two numeric id strings by length then lexically, which equals
// numeric order for non-negative integers without overflow.
func lessID(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
