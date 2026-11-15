package state

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tanq16/claudex/internal/fsutil"
)

type Project struct {
	RCName     string `json:"rcName"`
	Account    string `json:"account"`
	Connectors bool   `json:"connectors"`
	LastActive int64  `json:"lastActive,omitzero"`
}

type RemotePreset struct {
	Repo      string `json:"repo"`
	Path      string `json:"path,omitzero"`
	UpdatedAt int64  `json:"updatedAt"`
}

type State struct {
	Projects      map[string]Project      `json:"projects"`
	RemotePresets map[string]RemotePreset `json:"remotePresets"`
}

func Load(path string) (*State, error) {
	s := &State{}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(data, s); err != nil {
			return nil, fmt.Errorf("%s is not valid JSON: %w", path, err)
		}
	}
	if s.Projects == nil {
		s.Projects = map[string]Project{}
	}
	if s.RemotePresets == nil {
		s.RemotePresets = map[string]RemotePreset{}
	}
	return s, nil
}

func (s *State) Save(path string) error {
	data, err := json.Marshal(s, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, append(data, '\n'), 0o600)
}
