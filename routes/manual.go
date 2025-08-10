package routes

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"kommtkevinonline.de/queue"
	"kommtkevinonline.de/twitch"
)

func Manual(c *fiber.Ctx) error {

	vodId := c.Query("vodId")

	if vodId == "" {
		return c.SendString("No vodId provided")
	}

	video, err := twitch.GetVideoById(vodId)

	if err != nil {
		panic(err)
	}

	fmt.Printf("New Vod \"%s\" found.\n", video.ID)

	queue.AddToQueue(video)

	queue.Process()

	return c.SendString(vodId)
}
