package routes

import (
	"crypto/subtle"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"kommtkevinonline.de/queue"
	"kommtkevinonline.de/twitch"
)

func Manual(c *fiber.Ctx) error {
	expected := os.Getenv("MANUAL_API_TOKEN")

	// Fail closed: without a configured token the route stays unusable,
	// otherwise anyone reaching this port could trigger paid OpenAI calls.
	if expected == "" {
		return c.Status(fiber.StatusForbidden).SendString("MANUAL_API_TOKEN not configured")
	}

	provided := c.Query("token")
	if provided == "" {
		provided = c.Get("Authorization")
		provided = trimBearer(provided)
	}

	if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return c.Status(fiber.StatusUnauthorized).SendString("invalid token")
	}

	vodId := c.Query("vodId")

	if vodId == "" {
		return c.Status(fiber.StatusBadRequest).SendString("No vodId provided")
	}

	video, err := twitch.GetVideoById(vodId)

	if err != nil {
		log.Printf("manual: error fetching vod %q: %v", vodId, err)
		return c.Status(fiber.StatusBadGateway).SendString("failed to fetch vod")
	}

	log.Printf("manual: queueing vod %q", video.ID)

	queue.AddToQueue(video)

	go queue.Process()

	return c.SendString("queued " + vodId)
}

func trimBearer(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return header
}
