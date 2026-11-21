package server

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/claudex/internal/state"
	"github.com/tanq16/claudex/internal/workspace"
)

func (s *Server) handleTrack(r *http.Request) (any, error) {
	name := r.PathValue("name")
	var req struct {
		ForeignChoice string `json:"foreignChoice"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	dir, err := s.dirOf(name)
	if err != nil {
		return nil, err
	}
	account, err := s.resolveAccount("")
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	if _, tracked := st.Projects[dir]; tracked {
		return nil, fail(http.StatusConflict, "%s is already tracked", name)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, fail(http.StatusNotFound, "%s is not a directory under %s", name, s.abbrev(s.root))
	}
	if foreign := workspace.Foreign(dir); len(foreign) > 0 {
		if req.ForeignChoice != "preserve" && req.ForeignChoice != "clean" {
			return nil, fail(http.StatusConflict, "%s holds files claudex did not create (%s); choose preserve or clean", name, strings.Join(foreign, ", "))
		}
		if zipped, err := workspace.ClearForeign(dir, foreign, req.ForeignChoice == "preserve"); err != nil {
			if zipped != "" {
				return nil, fmt.Errorf("the old files are zipped in %s: %w", zipped, err)
			}
			return nil, err
		}
	}

	st.Projects[dir] = state.Project{Account: account, Connectors: true}
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	log.Info().Str("project", name).Msg("tracking started")
	return map[string]string{"name": name}, nil
}

func (s *Server) handleSetConnectors(r *http.Request) (any, error) {
	var req struct {
		Connectors bool `json:"connectors"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if err := s.storeConnectors(r.PathValue("name"), req.Connectors); err != nil {
		return nil, err
	}
	return s.handleRestart(r)
}

func (s *Server) storeConnectors(name string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return err
	}
	dir, p, err := s.tracked(st, name)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(s.projects(st), func(v projectView) bool { return v.Name == name && v.Active }) {
		return fail(http.StatusConflict, "%s is not running", name)
	}
	p.Connectors = on
	st.Projects[dir] = p
	return s.saveState(st)
}
