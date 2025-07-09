package templates

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Discover recorre dirRoot y devuelve archivos cuyo nombre coincida
// con los sufijos dados (*.html, *.gotpl, etc.). Si no hay sufijos, trae todo.
func Discover(dirRoot string, suffixes ...string) []string {
	return DiscoverFS(fs.FS(os.DirFS(dirRoot)), suffixes...)
}

func DiscoverFS(dirRoot fs.FS, suffixes ...string) []string {
	if len(suffixes) == 0 {
		suffixes = []string{"*"}
	}
	seen := make(map[string]struct{})
	var out []string

	_ = fs.WalkDir(dirRoot, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		for _, pat := range suffixes {
			if !strings.Contains(pat, "*") {
				pat = "*" + pat
			}
			if ok, _ := filepath.Match(pat, base); ok {
				if _, dup := seen[path]; !dup {
					seen[path] = struct{}{}
					out = append(out, path)
				}
				break
			}
		}
		return nil
	})
	sort.Strings(out)

	return out
}
