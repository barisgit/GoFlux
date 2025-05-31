package chi

import (
	"embed"
	"net/http"

	"github.com/barisgit/goflux/goflux"
)

// StaticHandler creates a Chi handler using the shared static logic
func StaticHandler(assets embed.FS, config goflux.StaticConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := goflux.ServeStaticFile(assets, config, r.URL.Path)

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
