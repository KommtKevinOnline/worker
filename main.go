package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"go.mau.fi/whatsmeow"
	"kommtkevinonline.de/database"
	"kommtkevinonline.de/routes"
	twitchLib "kommtkevinonline.de/twitch"
	"kommtkevinonline.de/whatsapp"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	godotenv.Load()

	db := database.GetConnection()

	database.RunMigrations(embedMigrations, db.DB)

	twitchLib.SetupOauth()

	var whatsappClient *whatsmeow.Client
	var err error

	if os.Getenv("INTEGRATION_WHATSAPP_ENABLED") == "true" {
		whatsappClient, err = whatsapp.Register(db.DB)
		if err != nil {
			println("Error registering whatsapp client: ", err)
		}
	}

	eventsubClient := twitchLib.RegisterEventSub()

	app := fiber.New()
	routes.RegisterRoutes(app)

	log.Fatal(app.Listen(":3000"))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if whatsappClient != nil {
			whatsappClient.Disconnect()
		}
		eventsubClient.Close()
		app.Shutdown()
		os.Exit(0)
	}()
}
