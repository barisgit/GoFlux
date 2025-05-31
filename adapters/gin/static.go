package gin

import (
	"embed"

	"github.com/barisgit/goflux/base"
	"github.com/gin-gonic/gin"
)

// StaticHandler creates a Gin handler using the shared static logic
func StaticHandler(assets embed.FS, config base.StaticConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := base.ServeStaticFile(assets, config, c.Request.URL.Path)

		if response.NotFound {
			c.AbortWithStatus(404)
			return
		}

		c.Header("Content-Type", response.ContentType)
		c.Header("Cache-Control", response.CacheControl)
		c.Data(response.StatusCode, response.ContentType, response.Body)
	}
}
