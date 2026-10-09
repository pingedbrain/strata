package markers

import (
	"bufio"
	"io"
	"io/fs"
)

// Scan reads markers from any stream. name is recorded on each Marker.
func Scan(r io.Reader, name string) ([]Marker, error) {
	var out []Marker
	sc := bufio.NewScanner(r)
	line := 0
	for sc.Scan() {
		line++
		for _, m := range re.FindAllStringSubmatch(sc.Text(), -1) {
			out = append(out, Marker{
				Kind:  kindOf(m[1]),
				ReqID: m[2],
				Hash:  m[3],
				File:  name,
				Line:  line,
			})
		}
	}
	return out, sc.Err()
}

// ScanDir walks fsys and scans every file not excluded by skip.
// skip(path, isDir) returning true prunes dirs and ignores files.
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
