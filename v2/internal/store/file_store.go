package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/zuhabul/ai-switch/v2/internal/model"
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Load() (model.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return model.NewState(), nil
	}

	b, err := os.ReadFile(s.path)
	if err != nil {
		return model.State{}, err
	}

	var state model.State
	if err := json.Unmarshal(b, &state); err != nil {
		return model.State{}, err
	}
	if state.Profiles == nil {
		state.Profiles = map[string]model.Profile{}
	}
	if state.Health == nil {
		state.Health = map[string]model.HealthSnapshot{}
	}
	if state.Leases == nil {
		state.Leases = map[string]model.Lease{}
	}
	if state.SecretBindings == nil {
		state.SecretBindings = map[string]map[string]string{}
	}
	if state.Policies == nil {
		state.Policies = []model.PolicyRule{}
	}
	return state, nil
}

func (s *FileStore) Save(state model.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
