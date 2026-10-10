package markers

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
)

// Scan reads markers from any stream. name is recorded on each Marker.
// Files with lines beyond 1 MiB (lockfiles, minified bundles, JSON
// captures) can't host markers — they're skipped, not fatal.
// @spec markers/scan
func Scan(r io.Reader, name string) ([]Marker, error) {
	var out []Marker
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		b := sc.Bytes()
		if bytes.IndexByte(b, 0) >= 0 {
			return nil, nil // binary file — no markers
		}
		line++
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			out = append(out, Marker{
				Kind:  kindOf(m[1]),
				ReqID: m[2],
				Hash:  m[3],
				File:  name,
				Line:  line,
			})
		}
	}
	if err := sc.Err(); err != nil && err != bufio.ErrTooLong {
		return out, err
	}
	return out, nil
}

// ScanDir walks fsys and scans every file not excluded by skip.
// skip(path, isDir) returning true prunes dirs and ignores files.
// @spec markers/scan-dir
func ScanDir(fsys fs.FS, skip func(path string, isDir bool) bool) ([]Marker, error) {
	var out []Marker
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip != nil && skip(p, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		f, err := fsys.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		ms, err := Scan(f, p)
		out = append(out, ms...)
		return err
	})
	return out, err
}
