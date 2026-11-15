package server

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/claudex/internal/accounts"
	"github.com/tanq16/claudex/internal/paths"
	"github.com/tanq16/claudex/internal/schema"
)

const schemaTimeout = 30 * time.Second

type defaultRow struct {
	schema.Entry
	Active   bool `json:"active"`
	Value    any  `json:"value"`
	InSchema bool `json:"inSchema"`
}

func (s *Server) loadDefaults() (*accounts.Defaults, error) {
	return accounts.LoadDefaults(paths.DefaultsFile(s.home))
}

func (s *Server) catalog(ctx context.Context) (schema.Catalog, error) {
	path := paths.SettingsSchemaFile(s.home)
	c, err := schema.Load(path)
	if errors.Is(err, fs.ErrNotExist) && s.schemaTried.CompareAndSwap(false, true) {
		ctx, cancel := context.WithTimeout(ctx, schemaTimeout)
		defer cancel()
		if c, err = schema.Refresh(ctx, path); err != nil {
			log.Warn().Err(err).Msg("could not fetch the Claude Code settings schema")
		}
	}
	return c, err
}

func valueKind(v any) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		return "number"
	}
	return "json"
}

func (s *Server) handleDefaults(r *http.Request) (any, error) {
	d, err := s.loadDefaults()
	if err != nil {
		return nil, err
	}
	c, schemaErr := s.catalog(r.Context())

	rows := []defaultRow{}
	for _, e := range c.Entries {
		if accounts.Owned(e.Key) {
			continue
		}
		v, active := d.Settings[e.Key]
		rows = append(rows, defaultRow{Entry: e, Active: active, Value: v, InSchema: true})
	}
	for _, key := range slices.Sorted(maps.Keys(d.Settings)) {
		if _, ok := c.Find(key); !ok {
			v := d.Settings[key]
			rows = append(rows, defaultRow{Entry: schema.Entry{Key: key, Kind: valueKind(v)}, Active: true, Value: v})
		}
	}
	slices.SortFunc(rows, func(a, b defaultRow) int { return strings.Compare(a.Key, b.Key) })

	status := map[string]any{"url": schema.URL, "fetchedAt": millis(c.FetchedAt), "keys": len(c.Entries)}
	if schemaErr != nil {
		msg := schemaErr.Error()
		if errors.Is(schemaErr, fs.ErrNotExist) {
			msg = "No schema fetched yet"
		}
		status["error"] = msg
	}
	return map[string]any{
		"rows":   rows,
		"owned":  slices.Sorted(maps.Keys(accounts.OwnedSettings(""))),
		"schema": status,
	}, nil
}

func (s *Server) handleSetDefault(r *http.Request) (any, error) {
	var req struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	req.Key = strings.TrimSpace(req.Key)
	if c, err := schema.Load(paths.SettingsSchemaFile(s.home)); err == nil {
		if e, ok := c.Find(req.Key); ok {
			if err := e.Check(req.Value); err != nil {
				return nil, fail(http.StatusBadRequest, "%v", err)
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	d, err := s.loadDefaults()
	if err != nil {
		return nil, err
	}
	if err := d.Set(req.Key, req.Value); err != nil {
		return nil, fail(http.StatusBadRequest, "%v", err)
	}
	if err := d.Save(paths.DefaultsFile(s.home)); err != nil {
		return nil, err
	}
	return map[string]string{"key": req.Key}, nil
}

func (s *Server) handleResetDefault(r *http.Request) (any, error) {
	key := r.URL.Query().Get("key")
	s.mu.Lock()
	defer s.mu.Unlock()
	d, err := s.loadDefaults()
	if err != nil {
		return nil, err
	}
	if !d.Reset(key) {
		return nil, fail(http.StatusNotFound, "%s is not an active key", key)
	}
	if err := d.Save(paths.DefaultsFile(s.home)); err != nil {
		return nil, err
	}
	return map[string]string{"key": key}, nil
}

func (s *Server) handleRefreshSchema(r *http.Request) (any, error) {
	ctx, cancel := context.WithTimeout(r.Context(), schemaTimeout)
	defer cancel()
	c, err := schema.Refresh(ctx, paths.SettingsSchemaFile(s.home))
	if err != nil {
		return nil, fail(http.StatusBadGateway, "fetching the settings schema failed: %v", err)
	}
	s.schemaTried.Store(true)
	return map[string]any{"fetchedAt": millis(c.FetchedAt), "keys": len(c.Entries)}, nil
}
