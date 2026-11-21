package server

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/claudex/internal/convo"
)

func (s *Server) handleHistory(r *http.Request) (any, error) {
	name := r.PathValue("name")
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, p, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	conv := convo.Conversation{Messages: []convo.Message{}}
	anchor, live := s.host.History(name)
	if live && anchor.Before == "" && !anchor.Whole {
		return conv, nil
	}
	if conv, err = convo.History(p.Account, dir, anchor.Before); err != nil {
		return nil, err
	}
	for i := range conv.Messages {
		m := &conv.Messages[i]
		if m.Message, err = s.images.Rewrite(m.Message); err != nil {
			log.Warn().Err(err).Str("project", name).Msg("could not store an image from the transcript")
		}
		if len(m.ToolUseResult) > 0 {
			if m.ToolUseResult, err = s.images.Rewrite(m.ToolUseResult); err != nil {
				log.Warn().Err(err).Str("project", name).Msg("could not store an image from the transcript")
			}
		}
	}
	return conv, nil
}
