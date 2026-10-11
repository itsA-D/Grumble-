package web

import (
	"io/fs"
	"net/http"

	webassets "github.com/OWNER/painboard/web"
)

// staticHandler serves the embedded assets under /static/.
func staticHandler() http.Handler {
	sub, err := fs.Sub(webassets.Static, "static")
	if err != nil {
		panic("web: static assets are not embedded: " + err.Error())
	}
	return http.StripPrefix("/static/", cacheStatic(http.FileServer(http.FS(sub))))
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}
