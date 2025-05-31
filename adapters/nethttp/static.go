package nethttp

import (
	"embed"
	"net/http"

	"github.com/barisgit/goflux/base"
)

// StaticHandler creates a net/http handler using the shared static logic
// Compatible with standard library mux, gorilla mux, and fasthttp (via adapter)
func StaticHandler(assets embed.FS, config base.StaticConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := base.ServeStaticFile(assets, config, r.URL.Path)

		if response.NotFound {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", response.ContentType)
		w.Header().Set("Cache-Control", response.CacheControl)
		w.WriteHeader(response.StatusCode)
		w.Write(response.Body)
	})
}

// StaticHandlerFunc creates a net/http HandlerFunc using the shared static logic
// Alternative function signature for cases where HandlerFunc is preferred
func StaticHandlerFunc(assets embed.FS, config base.StaticConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := base.ServeStaticFile(assets, config, r.URL.Path)

		if response.NotFound {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", response.ContentType)
		w.Header().Set("Cache-Control", response.CacheControl)
		w.WriteHeader(response.StatusCode)
		w.Write(response.Body)
	}
}
