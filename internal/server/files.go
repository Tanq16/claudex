package server

import (
	"bytes"
	"cmp"
	"errors"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	maxTextFile  = 2 << 20
	maxUploadMem = 32 << 20
)

var skippedDirs = []string{".git", "node_modules", ".venv"}

var imageExts = []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".bmp", ".avif"}

var langByExt = map[string]string{
	".go": "go", ".mod": "go", ".sum": "plaintext",
	".js": "javascript", ".mjs": "javascript", ".cjs": "javascript", ".jsx": "javascript",
	".ts": "typescript", ".tsx": "typescript", ".mts": "typescript", ".cts": "typescript",
	".json": "json", ".yaml": "yaml", ".yml": "yaml", ".toml": "ini", ".ini": "ini",
	".html": "html", ".htm": "html", ".xml": "xml", ".css": "css", ".scss": "scss",
	".sh": "bash", ".bash": "bash", ".zsh": "bash", ".py": "python", ".rb": "ruby",
	".rs": "rust", ".java": "java", ".kt": "kotlin", ".c": "c", ".h": "c", ".cpp": "cpp",
	".sql": "sql", ".lua": "lua", ".swift": "swift", ".tf": "hcl", ".hcl": "hcl",
	".txt": "plaintext", ".log": "plaintext", ".env": "bash", ".gitignore": "bash",
}

var langByName = map[string]string{
	"Makefile": "makefile", "Dockerfile": "dockerfile", "go.mod": "go", "go.sum": "plaintext",
}

type treeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"`
	Kind     string      `json:"kind,omitzero"`
	Target   string      `json:"target,omitzero"`
	Size     *int64      `json:"size,omitzero"`
	Children []*treeNode `json:"children,omitzero"`
}

type fileView struct {
	Path    string  `json:"path"`
	Kind    string  `json:"kind"`
	Lang    string  `json:"lang,omitzero"`
	Content *string `json:"content,omitzero"`
	Target  string  `json:"target,omitzero"`
	URL     string  `json:"url,omitzero"`
	Size    *int64  `json:"size,omitzero"`
}

func classify(name string, head []byte) (kind, lang string) {
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case ext == ".md" || ext == ".markdown" || ext == ".mdx":
		return "markdown", ""
	case slices.Contains(imageExts, ext):
		return "image", ""
	}
	if lang := cmp.Or(langByName[name], langByExt[ext]); lang != "" {
		return "code", lang
	}
	if head != nil && utf8.Valid(head) && !bytes.Contains(head, []byte{0}) {
		return "code", "plaintext"
	}
	return "binary", ""
}

func (s *Server) projectRoot(r *http.Request) (string, *os.Root, error) {
	name := r.PathValue("name")
	st, err := s.loadState()
	if err != nil {
		return "", nil, err
	}
	dir, _, err := s.tracked(st, name)
	if err != nil {
		return "", nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", nil, fail(http.StatusNotFound, "%s cannot be opened: %v", name, err)
	}
	return name, root, nil
}

func localPath(p string, allowRoot bool) (string, error) {
	if p == "" && allowRoot {
		return ".", nil
	}
	rel := filepath.FromSlash(p)
	if !filepath.IsLocal(rel) {
		return "", fail(http.StatusBadRequest, "%q is not a path inside the project", p)
	}
	return rel, nil
}

func (s *Server) handleTree(r *http.Request) (any, error) {
	name, root, err := s.projectRoot(r)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	node := &treeNode{Name: name, Path: "", Type: "dir"}
	if err := walkTree(root, ".", node); err != nil {
		return nil, err
	}
	return node, nil
}

func walkTree(root *os.Root, dir string, node *treeNode) error {
	entries, err := fs.ReadDir(root.FS(), dir)
	if err != nil {
		return err
	}
	node.Children = []*treeNode{}
	for _, e := range entries {
		rel := path.Join(dir, e.Name())
		child := &treeNode{Name: e.Name(), Path: rel}
		switch {
		case e.Type()&fs.ModeSymlink != 0:
			child.Type, child.Kind = "symlink", "symlink"
			child.Target, _ = root.Readlink(rel)
			child.Size = new(int64(len(child.Target)))
		case e.IsDir():
			if slices.Contains(skippedDirs, e.Name()) {
				continue
			}
			child.Type = "dir"
			if err := walkTree(root, rel, child); err != nil {
				return err
			}
		case e.Type().IsRegular():
			info, err := e.Info()
			if err != nil {
				continue
			}
			child.Type = "file"
			child.Kind, _ = classify(e.Name(), sniff(root, rel))
			child.Size = new(info.Size())
		default:
			continue
		}
		node.Children = append(node.Children, child)
	}
	slices.SortFunc(node.Children, func(a, b *treeNode) int {
		if (a.Type == "dir") != (b.Type == "dir") {
			if a.Type == "dir" {
				return -1
			}
			return 1
		}
		return cmp.Or(strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), strings.Compare(a.Name, b.Name))
	})
	return nil
}

func sniff(root *os.Root, rel string) []byte {
	if kind, _ := classify(rel, nil); kind != "binary" {
		return nil
	}
	f, err := root.Open(rel)
	if err != nil {
		return nil
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	return head[:n]
}

func rawURL(name, rel string) string {
	return "/api/projects/" + url.PathEscape(name) + "/raw?path=" + url.QueryEscape(rel)
}

func (s *Server) handleFile(r *http.Request) (any, error) {
	name, root, err := s.projectRoot(r)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	query := r.URL.Query().Get("path")
	rel, err := localPath(query, false)
	if err != nil {
		return nil, err
	}

	info, err := root.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fail(http.StatusNotFound, "%s not found", query)
	}
	if err != nil {
		return nil, fail(http.StatusBadRequest, "%s cannot be read inside the project: %v", query, err)
	}
	view := fileView{Path: query}
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		view.Kind = "symlink"
		view.Target, _ = root.Readlink(rel)
		return view, nil
	case info.IsDir():
		return nil, fail(http.StatusBadRequest, "%s is a directory", query)
	}

	size := info.Size()
	view.Size = &size
	kind, lang := classify(info.Name(), nil)
	if kind == "image" {
		view.Kind, view.URL = "image", rawURL(name, query)
		return view, nil
	}
	if size > maxTextFile {
		view.Kind = "binary"
		return view, nil
	}
	data, err := root.ReadFile(rel)
	if err != nil {
		return nil, err
	}
	if kind == "binary" {
		kind, lang = classify(info.Name(), data[:min(len(data), 512)])
	}
	if kind != "binary" && !utf8.Valid(data) {
		kind, lang = "binary", ""
	}
	view.Kind, view.Lang = kind, lang
	if kind != "binary" {
		view.Content = new(string(data))
	}
	return view, nil
}

func (s *Server) handleRaw(w http.ResponseWriter, r *http.Request) {
	_, root, err := s.projectRoot(r)
	if err != nil {
		writeError(w, statusOf(err), err.Error())
		return
	}
	defer root.Close()
	query := r.URL.Query().Get("path")
	rel, err := localPath(query, false)
	if err != nil {
		writeError(w, statusOf(err), err.Error())
		return
	}
	f, err := root.Open(rel)
	if err != nil {
		writeError(w, http.StatusNotFound, query+" not found")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		writeError(w, http.StatusBadRequest, query+" is not a file")
		return
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func (s *Server) handleCreateFile(r *http.Request) (any, error) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	_, root, err := s.projectRoot(r)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	rel, err := localPath(req.Path, false)
	if err != nil {
		return nil, err
	}
	if parent := filepath.Dir(rel); parent != "." {
		if err := root.MkdirAll(parent, 0o755); err != nil {
			return nil, fail(http.StatusBadRequest, "%v", err)
		}
	}
	f, err := root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return nil, fail(http.StatusConflict, "%s already exists", req.Path)
	}
	if err != nil {
		return nil, fail(http.StatusBadRequest, "%v", err)
	}
	if _, err := io.WriteString(f, req.Content); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return map[string]string{"path": req.Path}, nil
}

func (s *Server) handleUpload(r *http.Request) (any, error) {
	_, root, err := s.projectRoot(r)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if err := r.ParseMultipartForm(maxUploadMem); err != nil {
		return nil, fail(http.StatusBadRequest, "the upload is not a multipart form: %v", err)
	}
	defer r.MultipartForm.RemoveAll()

	folder, err := localPath(r.FormValue("folder"), true)
	if err != nil {
		return nil, err
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		return nil, fail(http.StatusBadRequest, "the upload carries no files")
	}
	if folder != "." {
		if err := root.MkdirAll(folder, 0o755); err != nil {
			return nil, fail(http.StatusBadRequest, "%v", err)
		}
	}
	for _, fh := range files {
		base := path.Base(strings.ReplaceAll(fh.Filename, `\`, "/"))
		if base == "." || base == ".." || base == "/" {
			return nil, fail(http.StatusBadRequest, "%q is not a file name", fh.Filename)
		}
		if err := saveUpload(root, filepath.Join(folder, base), fh); err != nil {
			return nil, err
		}
	}
	return map[string]int{"count": len(files)}, nil
}

func saveUpload(root *os.Root, rel string, fh *multipart.FileHeader) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fail(http.StatusBadRequest, "%v", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}
