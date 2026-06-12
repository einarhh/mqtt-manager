// Package plugins persists user-authored decoder plugins. A plugin is a small
// JavaScript module (see the README) that the frontend runs to decode matching
// MQTT payloads for display. The Go side only stores and serves the source; it
// never executes it.
package plugins

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Plugin is a decoder plugin. Source holds the JavaScript module text; it is
// read from / written to an individual .js file so plugins are shareable as
// plain files. The remaining fields are tracked in an index file.
type Plugin struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Filename string `json:"filename"` // basename within the plugins dir, e.g. "stigmesh.js"
	Enabled  bool   `json:"enabled"`
	Order    int    `json:"order"`
	Source   string `json:"source"`
}

// meta is the per-plugin record kept in the index file (everything but Source).
type meta struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Enabled  bool   `json:"enabled"`
	Order    int    `json:"order"`
}

// Store persists plugins to <UserConfigDir>/mqtt-manager/plugins/. Each plugin's
// source lives in its own .js file; an index.json tracks name/enabled/order.
type Store struct {
	mu    sync.Mutex
	dir   string
	index string
}

var slugRE = regexp.MustCompile(`[^a-z0-9._-]+`)

// NewStore returns a store backed by <UserConfigDir>/mqtt-manager/plugins/.
func NewStore() (*Store, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locating config dir: %w", err)
	}
	dir := filepath.Join(cfgDir, "mqtt-manager", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating plugins dir: %w", err)
	}
	return &Store{dir: dir, index: filepath.Join(dir, "index.json")}, nil
}

// List returns all plugins with their source, sorted by order then name. Loose
// .js files dropped into the directory by hand are discovered and appended
// (disabled by default) so plugins can be shared simply by copying a file in.
func (s *Store) List() ([]Plugin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list()
}

func (s *Store) list() ([]Plugin, error) {
	metas, err := s.readIndex()
	if err != nil {
		return nil, err
	}

	known := make(map[string]bool, len(metas))
	out := make([]Plugin, 0, len(metas))
	for _, m := range metas {
		known[m.Filename] = true
		src, err := os.ReadFile(filepath.Join(s.dir, m.Filename))
		if err != nil {
			if os.IsNotExist(err) {
				continue // file removed under us; drop from the listing
			}
			return nil, fmt.Errorf("reading plugin %s: %w", m.Filename, err)
		}
		out = append(out, Plugin{
			ID: m.ID, Name: m.Name, Filename: m.Filename,
			Enabled: m.Enabled, Order: m.Order, Source: string(src),
		})
	}

	// Discover loose .js files not present in the index.
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("reading plugins dir: %w", err)
	}
	maxOrder := len(out)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || known[name] || !strings.HasSuffix(name, ".js") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(s.dir, name))
		if err != nil {
			return nil, fmt.Errorf("reading plugin %s: %w", name, err)
		}
		out = append(out, Plugin{
			ID:       newID(),
			Name:     strings.TrimSuffix(name, ".js"),
			Filename: name,
			Enabled:  false,
			Order:    maxOrder,
			Source:   string(src),
		})
		maxOrder++
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Save inserts or updates a plugin, assigning an ID and filename if missing, and
// returns the stored value.
func (s *Store) Save(p Plugin) (Plugin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metas, err := s.readIndex()
	if err != nil {
		return p, err
	}
	if p.ID == "" {
		p.ID = newID()
	}
	if p.Filename == "" {
		p.Filename = s.uniqueFilename(p.Name, p.ID, metas)
	} else {
		p.Filename = sanitizeFilename(p.Filename)
	}

	if err := s.writeSource(p.Filename, p.Source); err != nil {
		return p, err
	}

	m := meta{ID: p.ID, Name: p.Name, Filename: p.Filename, Enabled: p.Enabled, Order: p.Order}
	replaced := false
	for i := range metas {
		if metas[i].ID == p.ID {
			// If the filename changed, remove the old file.
			if metas[i].Filename != p.Filename {
				_ = os.Remove(filepath.Join(s.dir, metas[i].Filename))
			}
			metas[i] = m
			replaced = true
			break
		}
	}
	if !replaced {
		m.Order = len(metas)
		p.Order = m.Order
		metas = append(metas, m)
	}
	return p, s.writeIndex(metas)
}

// Delete removes a plugin (index entry and .js file) by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metas, err := s.readIndex()
	if err != nil {
		return err
	}
	out := metas[:0]
	for _, m := range metas {
		if m.ID == id {
			_ = os.Remove(filepath.Join(s.dir, m.Filename))
			continue
		}
		out = append(out, m)
	}
	return s.writeIndex(out)
}

func (s *Store) readIndex() ([]meta, error) {
	data, err := os.ReadFile(s.index)
	if os.IsNotExist(err) {
		return []meta{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading plugin index: %w", err)
	}
	var list []meta
	if len(data) > 0 {
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("parsing plugin index: %w", err)
		}
	}
	return list, nil
}

func (s *Store) writeIndex(list []meta) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding plugin index: %w", err)
	}
	return atomicWrite(s.index, data, 0o600)
}

func (s *Store) writeSource(filename, source string) error {
	return atomicWrite(filepath.Join(s.dir, filename), []byte(source), 0o644)
}

// uniqueFilename derives a "<slug>.js" name from the plugin name, falling back
// to the ID, and disambiguates against existing index entries.
func (s *Store) uniqueFilename(name, id string, metas []meta) string {
	base := slugify(name)
	if base == "" {
		base = "plugin-" + id
	}
	taken := make(map[string]bool, len(metas))
	for _, m := range metas {
		taken[m.Filename] = true
	}
	candidate := base + ".js"
	for i := 2; taken[candidate]; i++ {
		candidate = fmt.Sprintf("%s-%d.js", base, i)
	}
	return candidate
}

func slugify(name string) string {
	s := slugRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	return strings.Trim(s, "-._")
}

// sanitizeFilename strips any path components and enforces a .js extension.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = slugRE.ReplaceAllString(strings.ToLower(name), "-")
	name = strings.Trim(name, "-._")
	if name == "" {
		name = "plugin"
	}
	if !strings.HasSuffix(name, ".js") {
		name += ".js"
	}
	return name
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	return os.Rename(tmp, path)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
