package httpd

import (
	"io/fs"
	"net/http"
	"strings"
)

// Serve start een HTTP-server op addr met handler h.
func Serve(addr string, h http.Handler) error {
	return http.ListenAndServe(addr, h)
}

// Router combineert de API-mux met de statische SPA (fallback naar index.html).
func Router(api http.Handler, web fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(web))
	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA: onbekende non-asset paden -> index.html
		if _, err := fs.Stat(web, strings.TrimPrefix(r.URL.Path, "/")); err != nil && r.URL.Path != "/" {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
	return mux
}
