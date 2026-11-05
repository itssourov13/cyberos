package v1

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
)

// The production frontend is the Vite build of ./frontend, copied into
// ./static by `make web` (see docs/DEPLOYMENT.md). Only PLACEHOLDER is
// committed; "all:" keeps the directive valid before the first build.
//
//go:embed all:static
var uiFS embed.FS

var inlineScript = regexp.MustCompile(`(?s)<script(?:\s[^>]*)?>(.*?)</script>`)

// spaHandler serves the embedded single-page app.
//
//   - /api/* never falls through to index.html (JSON 404).
//   - Existing files are served with content-type by extension; hashed build
//     assets are immutable-cached, everything else revalidates.
//   - Unknown extension-less paths get index.html (client-side routes);
//     unknown paths with a file extension get a real 404.
//   - Paths are validated with fs.ValidPath, so traversal cannot leave the FS.
type spaHandler struct {
	fsys fs.FS
	csp  string
}

func newSPAHandler(fsys fs.FS) http.Handler {
	h := &spaHandler{fsys: fsys}
	h.csp = buildCSP(fsys)
	return h
}

func uiFileServer() http.Handler {
	sub, err := fs.Sub(uiFS, "static")
	if err != nil {
		return http.NotFoundHandler()
	}
	return newSPAHandler(sub)
}

// buildCSP allows same-origin resources plus the exact inline bootstrap
// scripts present in index.html (by SHA-256), without 'unsafe-inline' scripts.
func buildCSP(fsys fs.FS) string {
	scriptSrc := "'self'"
	if b, err := fs.ReadFile(fsys, "index.html"); err == nil {
		for _, m := range inlineScript.FindAllSubmatch(b, -1) {
			if len(bytes.TrimSpace(m[1])) == 0 {
				continue
			}
			sum := sha256.Sum256(m[1])
			scriptSrc += " 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
		}
	}
	return "default-src 'self'; script-src " + scriptSrc +
		"; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; " +
		"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if p == "/api" || strings.HasPrefix(p, "/api/") {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hdr := w.Header()
	hdr.Set("X-Content-Type-Options", "nosniff")
	hdr.Set("Referrer-Policy", "no-referrer")
	hdr.Set("X-Frame-Options", "DENY")
	hdr.Set("Content-Security-Policy", h.csp)

	clean := path.Clean("/" + p)
	name := strings.TrimPrefix(clean, "/")
	if name == "" {
		name = "index.html"
	}
	if !fs.ValidPath(name) {
		http.NotFound(w, r)
		return
	}
	if st, err := fs.Stat(h.fsys, name); err == nil && !st.IsDir() && name != "PLACEHOLDER" {
		h.serveFile(w, r, name, st.ModTime())
		return
	}
	// Missing asset (has an extension): a real 404, never the SPA shell.
	if path.Ext(name) != "" && name != "index.html" {
		http.NotFound(w, r)
		return
	}
	if _, err := fs.Stat(h.fsys, "index.html"); err != nil {
		hdr.Set("Content-Type", "text/plain; charset=utf-8")
		hdr.Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("CYBEROS frontend is not built into this binary. Run `make web && make build`.\n"))
		return
	}
	h.serveFile(w, r, "index.html", time.Time{})
}

func (h *spaHandler) serveFile(w http.ResponseWriter, r *http.Request, name string, mod time.Time) {
	f, err := h.fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	rs, ok := f.(interface {
		Read([]byte) (int, error)
		Seek(int64, int) (int64, error)
	})
	if !ok {
		http.NotFound(w, r)
		return
	}
	hdr := w.Header()
	if strings.HasPrefix(name, "assets/") {
		hdr.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		hdr.Set("Cache-Control", "no-cache")
	}
	http.ServeContent(w, r, name, mod, rs)
}
