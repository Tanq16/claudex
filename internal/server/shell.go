package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/tanq16/claudex/internal/accounts"
	"github.com/tanq16/claudex/internal/shell"
)

func (s *Server) handleShell(r *http.Request) (any, error) {
	var req struct {
		Command string `json:"command"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Command) == "" {
		return nil, fail(http.StatusBadRequest, "command is required")
	}
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, p, err := s.tracked(st, r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	return shell.Run(r.Context(), shell.Config{
		Shell:   os.Getenv("SHELL"),
		Dir:     dir,
		Env:     accounts.Env(p.Account, s.launcher.DefaultAccount),
		Command: req.Command,
	})
}
