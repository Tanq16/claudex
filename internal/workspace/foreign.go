package workspace

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const OldFilesZip = "old-ai-files.zip"

var foreignCandidates = []string{MCPFile, ClaudeDir, AgentsDir, ClaudeFile, AgentsFile}

func Owned(root string) bool {
	open, _ := markers("")
	if !strings.Contains(readFile(filepath.Join(root, AgentsFile)), open) {
		return false
	}
	if target, err := os.Readlink(filepath.Join(root, ClaudeFile)); err != nil || target != AgentsFile {
		return false
	}
	target, err := os.Readlink(filepath.Join(root, ClaudeDir, SkillsDir))
	return err == nil && target == filepath.Join("..", AgentsDir, SkillsDir)
}

func Foreign(root string) []string {
	if Owned(root) {
		return nil
	}
	var found []string
	for _, rel := range foreignCandidates {
		info, err := os.Lstat(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		if info.IsDir() {
			rel += "/"
		}
		found = append(found, rel)
	}
	return found
}

func RemovePaths(root string, rels []string) error {
	for _, rel := range rels {
		if err := os.RemoveAll(filepath.Join(root, strings.TrimSuffix(rel, "/"))); err != nil {
			return err
		}
	}
	return nil
}

func ZipPaths(root string, rels []string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, rel := range rels {
		start := filepath.Join(root, strings.TrimSuffix(rel, "/"))
		if err := filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			return addToZip(zw, root, path, d)
		}); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addToZip(zw *zip.Writer, root, path string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	name, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = filepath.ToSlash(name)

	switch {
	case d.IsDir():
		hdr.Name += "/"
		_, err := zw.CreateHeader(hdr)
		return err
	case d.Type()&fs.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, target)
		return err
	case !d.Type().IsRegular():
		return nil
	}

	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func WriteOldFiles(root string, data []byte) error {
	return writeFileAtomic(filepath.Join(root, AgentsDir, OldFilesZip), data, projectModes.file)
}
