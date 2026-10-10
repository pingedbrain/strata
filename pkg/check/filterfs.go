package check

import (
	"io/fs"
	"path"
)

// filteredFS wraps an fs.FS and hides every path the skip predicate
// rejects — specs under ignored dirs never reach the adapters, and
// ignored files are invisible to marker/symbol scans.
type filteredFS struct {
	fsys fs.FS
	skip func(string, bool) bool
}

func (f filteredFS) skipPath(p string, isDir bool) bool {
	return p != "." && f.skip != nil && f.skip(p, isDir)
}

func (f filteredFS) Open(name string) (fs.File, error) {
	if f.skipPath(name, true) || f.skipPath(name, false) {
		return nil, fs.ErrNotExist
	}
	return f.fsys.Open(name)
}

func (f filteredFS) Stat(name string) (fs.FileInfo, error) {
	if f.skipPath(name, true) || f.skipPath(name, false) {
		return nil, fs.ErrNotExist
	}
	return fs.Stat(f.fsys, name)
}

func (f filteredFS) ReadFile(name string) ([]byte, error) {
	if f.skipPath(name, false) {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(f.fsys, name)
}

func (f filteredFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.fsys, name)
	if err != nil {
		return nil, err
	}
	out := entries[:0]
	for _, e := range entries {
		p := e.Name()
		if name != "." {
			p = name + "/" + p
		}
		if !f.skipPath(p, e.IsDir()) {
			out = append(out, e)
		}
	}
	return out, nil
}

// Glob resolves pattern over the filtered view. No GlobFS delegation —
// a manual walk keeps hidden paths out.
func (f filteredFS) Glob(pattern string) ([]string, error) {
	var out []string
	err := fs.WalkDir(f, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if ok, _ := path.Match(pattern, p); ok {
				out = append(out, p)
			}
		}
		return nil
	})
	return out, err
}

var (
	_ fs.FS         = filteredFS{}
	_ fs.ReadDirFS  = filteredFS{}
	_ fs.ReadFileFS = filteredFS{}
	_ fs.StatFS     = filteredFS{}
	_ fs.GlobFS     = filteredFS{}
)
