package schema

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"maps"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/tanq16/claudex/internal/fsutil"
)

const URL = "https://json.schemastore.org/claude-code-settings.json"

const maxSize = 16 << 20

var client = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

type Entry struct {
	Key         string         `json:"key"`
	Kind        string         `json:"kind"`
	Enum        []string       `json:"enum,omitzero"`
	Description string         `json:"description,omitzero"`
	Default     jsontext.Value `json:"default,omitzero"`
}

type Catalog struct {
	FetchedAt time.Time
	Entries   []Entry
}

func (c Catalog) Find(key string) (Entry, bool) {
	i := slices.IndexFunc(c.Entries, func(e Entry) bool { return e.Key == key })
	if i < 0 {
		return Entry{}, false
	}
	return c.Entries[i], true
}

func (e Entry) Check(v any) error {
	switch e.Kind {
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%s takes true or false", e.Key)
		}
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("%s takes a string", e.Key)
		}
	case "number", "integer":
		n, ok := v.(float64)
		if !ok {
			return fmt.Errorf("%s takes a number", e.Key)
		}
		if e.Kind == "integer" && n != math.Trunc(n) {
			return fmt.Errorf("%s takes a whole number", e.Key)
		}
	case "enum":
		if s, ok := v.(string); !ok || !slices.Contains(e.Enum, s) {
			return fmt.Errorf("%s takes one of %v", e.Key, e.Enum)
		}
	}
	return nil
}

func Load(path string) (Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Catalog{}, err
	}
	entries, err := parse(data)
	if err != nil {
		return Catalog{}, fmt.Errorf("%s: %w", path, err)
	}
	return Catalog{FetchedAt: info.ModTime(), Entries: entries}, nil
}

func Refresh(ctx context.Context, path string) (Catalog, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, nil)
	if err != nil {
		return Catalog{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Catalog{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Catalog{}, fmt.Errorf("%s returned %s", URL, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return Catalog{}, err
	}
	entries, err := parse(data)
	if err != nil {
		return Catalog{}, fmt.Errorf("%s: %w", URL, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Catalog{}, err
	}
	if err := fsutil.WriteFileAtomic(path, data, 0o600); err != nil {
		return Catalog{}, err
	}
	return Catalog{FetchedAt: time.Now(), Entries: entries}, nil
}

type property struct {
	Type        jsontext.Value      `json:"type"`
	Enum        []jsontext.Value    `json:"enum"`
	Description string              `json:"description"`
	Default     jsontext.Value      `json:"default"`
	Properties  map[string]property `json:"properties"`
}

func parse(data []byte) ([]Entry, error) {
	var doc struct {
		Properties map[string]property `json:"properties"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("not a settings schema: %w", err)
	}
	var entries []Entry
	flatten("", doc.Properties, &entries)
	if len(entries) == 0 {
		return nil, fmt.Errorf("the schema lists no settings")
	}
	return entries, nil
}

func flatten(prefix string, props map[string]property, out *[]Entry) {
	for _, name := range slices.Sorted(maps.Keys(props)) {
		if name == "$schema" {
			continue
		}
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		p := props[name]
		if len(p.Properties) > 0 && slices.Contains(p.types(), "object") {
			flatten(key, p.Properties, out)
			continue
		}
		*out = append(*out, Entry{Key: key, Kind: p.kind(), Enum: p.enumStrings(), Description: p.Description, Default: p.Default})
	}
}

func (p property) types() []string {
	var one string
	if json.Unmarshal(p.Type, &one) == nil {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(p.Type, &many) == nil {
		return many
	}
	return nil
}

func (p property) enumStrings() []string {
	values := make([]string, 0, len(p.Enum))
	for _, raw := range p.Enum {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return nil
		}
		values = append(values, s)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (p property) kind() string {
	if p.enumStrings() != nil {
		return "enum"
	}
	scalar := slices.DeleteFunc(p.types(), func(t string) bool { return t == "null" })
	if len(scalar) == 1 && slices.Contains([]string{"boolean", "string", "number", "integer"}, scalar[0]) {
		return scalar[0]
	}
	return "json"
}
