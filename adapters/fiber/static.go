package fiber

import (
	"embed"

	"github.com/barisgit/goflux/base"
	"github.com/gofiber/fiber/v2"
)

// StaticHandler creates a Fiber handler using the shared static logic
func StaticHandler(assets embed.FS, config base.StaticConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		response := base.ServeStaticFile(assets, config, c.Path())

		if response.NotFound {
			return c.SendStatus(404)
		}

		c.Set("Content-Type", response.ContentType)
		c.Set("Cache-Control", response.CacheControl)
		c.Status(response.StatusCode)
		return c.Send(response.Body)
	}
}
