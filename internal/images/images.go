package images

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

const (
	maxAge      = 7 * 24 * time.Hour
	thumbHeight = 288
)

var (
	fileName = regexp.MustCompile(`^[0-9a-f]{64}\.[a-z0-9]+$`)
	lenient  = json.JoinOptions(jsontext.AllowInvalidUTF8(true), jsontext.AllowDuplicateNames(true))
	marker   = []byte(`"base64"`)
)

var extensions = map[string]string{
	"image/png":       "png",
	"image/jpeg":      "jpg",
	"image/gif":       "gif",
	"image/webp":      "webp",
	"image/svg+xml":   "svg",
	"application/pdf": "pdf",
}

type Store struct {
	dir    string
	prefix string
}

func New(dir, prefix string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > maxAge {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return &Store{dir: dir, prefix: prefix}, nil
}

func (s *Store) Path(name string) (string, bool) {
	if !fileName.MatchString(name) {
		return "", false
	}
	path := filepath.Join(s.dir, name)
	_, err := os.Stat(path)
	return path, err == nil
}

func (s *Store) Thumb(path string) (string, error) {
	thumb := strings.TrimSuffix(path, filepath.Ext(path)) + ".thumb.png"
	if _, err := os.Stat(thumb); err == nil {
		return thumb, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if errors.Is(err, image.ErrFormat) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	if (format != "png" && format != "jpeg") || cfg.Height <= thumbHeight {
		return path, nil
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	dst := image.NewRGBA(image.Rect(0, 0, max(1, cfg.Width*thumbHeight/cfg.Height), thumbHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return "", err
	}
	if err := writeOnce(thumb, buf.Bytes()); err != nil {
		return "", err
	}
	return thumb, nil
}

func (s *Store) Rewrite(raw []byte) ([]byte, error) {
	if !bytes.Contains(raw, marker) {
		return raw, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v, lenient); err != nil {
		return raw, err
	}
	changed, err := s.walk(v)
	if err != nil || !changed {
		return raw, err
	}
	return json.Marshal(v, lenient)
}

func (s *Store) walk(v any) (bool, error) {
	changed := false
	switch t := v.(type) {
	case map[string]any:
		if data, ok := t["data"].(string); ok && t["type"] == "base64" {
			url, err := s.save(t["media_type"], data)
			if err != nil {
				return false, err
			}
			t["type"], t["url"] = "url", url
			delete(t, "data")
			changed = true
		}
		if data, ok := t["base64"].(string); ok {
			url, err := s.save(t["type"], data)
			if err != nil {
				return false, err
			}
			t["url"] = url
			delete(t, "base64")
			changed = true
		}
		for _, child := range t {
			c, err := s.walk(child)
			if err != nil {
				return false, err
			}
			changed = changed || c
		}
	case []any:
		for _, child := range t {
			c, err := s.walk(child)
			if err != nil {
				return false, err
			}
			changed = changed || c
		}
	}
	return changed, nil
}

func (s *Store) save(mediaType any, encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	mt, _ := mediaType.(string)
	ext, ok := extensions[mt]
	if !ok {
		ext = "bin"
	}
	name := hex.EncodeToString(sum[:]) + "." + ext
	path := filepath.Join(s.dir, name)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := writeOnce(path, data); err != nil {
			return "", err
		}
	} else if err == nil {
		now := time.Now()
		os.Chtimes(path, now, now)
	}
	return s.prefix + name, nil
}

func writeOnce(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return err
	}
	return os.Rename(f.Name(), path)
}
