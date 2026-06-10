package profiles

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Store persists connection profiles to a JSON file in the user config dir.
type Store struct {
	mu   sync.Mutex
	path string
}

// NewStore returns a store backed by <UserConfigDir>/mqtt-manager/profiles.json.
func NewStore() (*Store, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locating config dir: %w", err)
	}
	dir := filepath.Join(cfgDir, "mqtt-manager")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating config dir: %w", err)
	}
	return &Store{path: filepath.Join(dir, "profiles.json")}, nil
}

// List returns all saved profiles sorted by name.
func (s *Store) List() ([]ConnectionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

// Save inserts or updates a profile, assigning an ID if missing.
func (s *Store) Save(p ConnectionProfile) (ConnectionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.read()
	if err != nil {
		return p, err
	}
	if p.ID == "" {
		p.ID = newID()
	}
	replaced := false
	for i := range list {
		if list[i].ID == p.ID {
			list[i] = p
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, p)
	}
	return p, s.write(list)
}

// Delete removes a profile by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.read()
	if err != nil {
		return err
	}
	out := list[:0]
	for _, p := range list {
		if p.ID != id {
			out = append(out, p)
		}
	}
	return s.write(out)
}

// read loads the profile list, returning an empty slice if the file is absent.
func (s *Store) read() ([]ConnectionProfile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []ConnectionProfile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading profiles: %w", err)
	}
	var list []ConnectionProfile
	if len(data) > 0 {
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("parsing profiles: %w", err)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, nil
}

// write atomically persists the profile list.
func (s *Store) write(list []ConnectionProfile) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding profiles: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing profiles: %w", err)
	}
	return os.Rename(tmp, s.path)
}

// newID returns a random hex identifier.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
