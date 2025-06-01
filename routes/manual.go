package routes

import "github.com/gofiber/fiber/v2"

func Manual(c *fiber.Ctx) error {
	return c.SendString("Hello, World!")
}
