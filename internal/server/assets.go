package server

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var compressible = []string{".css", ".html", ".js", ".json", ".mjs", ".svg", ".wasm"}

type asset struct {
	data, gz    []byte
	etag, gzTag string
}

func loadAssets(fsys fs.FS) (map[string]func() (*asset, error), error) {
	assets := map[string]func() (*asset, error){}
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		assets[name] = sync.OnceValues(func() (*asset, error) {
			data, err := fs.ReadFile(fsys, name)
			if err != nil {
				return nil, err
			}
			sum := sha256.Sum256(data)
			hash := hex.EncodeToString(sum[:12])
			a := &asset{data: data, etag: `"` + hash + `"`, gzTag: `"` + hash + `-gz"`}
			if slices.Contains(compressible, path.Ext(name)) {
				a.gz = gzipped(data)
			}
			return a, nil
		})
		return nil
	})
	return assets, err
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/static/")
	load, ok := s.assets[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	a, err := load()
	if err != nil {
		log.Error().Err(err).Str("path", r.URL.Path).Msg("could not read a static asset")
		http.Error(w, "the asset could not be read", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Cache-Control", "no-cache")
	body, tag := a.data, a.etag
	if a.gz != nil {
		h.Add("Vary", "Accept-Encoding")
		if acceptsGzip(r) {
			body, tag = a.gz, a.gzTag
			h.Set("Content-Encoding", "gzip")
		}
	}
	h.Set("ETag", tag)
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(body))
}

func acceptsGzip(r *http.Request) bool {
	for part := range strings.SplitSeq(r.Header.Get("Accept-Encoding"), ",") {
		coding, params, _ := strings.Cut(part, ";")
		if strings.TrimSpace(coding) == "gzip" {
			return strings.ReplaceAll(params, " ", "") != "q=0"
		}
	}
	return false
}

func gzipped(data []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(data)
	zw.Close()
	return buf.Bytes()
}
