package fiber

import (
	"embed"

	"github.com/barisgit/goflux"
	"github.com/gofiber/fiber/v2"
)

// StaticHandler creates a Fiber handler using the shared static logic
func StaticHandler(assets embed.FS, config goflux.StaticConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		response := goflux.ServeStaticFile(assets, config, c.Path())

		if response.NotFound {
			return c.SendStatus(404)
		}

		c.Set("Content-Type", response.ContentType)
		c.Set("Cache-Control", response.CacheControl)
		c.Status(response.StatusCode)
		return c.Send(response.Body)
	}
}
