package static

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strings"

	"github.com/kamil-koziol/pingo/internal/web/components/ui/utils"
)

//go:embed css js fonts
var FS embed.FS

// hashes maps an embedded file path (e.g. "css/output.css")
var hashes = map[string]string{}

func init() {
	_ = fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := FS.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		hashes[path] = hex.EncodeToString(sum[:])[:12]
		return nil
	})

	// Override the templui cache busting
	utils.ScriptURL = func(p string) string {
		return URL(strings.TrimPrefix(p, "/static/"))
	}
}

// URL returns the public URL for an embedded file, with its content hash
// added, e.g. URL("css/output.css") -> "/static/css/output.css?v=3fa9c1b2d4e5".
func URL(name string) string {
	if h, ok := hashes[name]; ok {
		return "/static/" + name + "?v=" + h
	}
	return "/static/" + name
}

func Handler() http.Handler {
	files := http.FileServerFS(FS)

	return http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, known := hashes[r.URL.Path]
		if known {
			w.Header().Set("ETag", `"`+h+`"`)
		}

		if known && r.URL.Query().Get("v") == h {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		files.ServeHTTP(w, r)
	}))
}
