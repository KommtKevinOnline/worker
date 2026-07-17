package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	go twitchLib.SetupOauth()

	var whatsappClient *whatsmeow.Client
	var err error

	if os.Getenv("INTEGRATION_WHATSAPP_ENABLED") == "true" {
		whatsappClient, err = whatsapp.Register(db.DB)
		if err != nil {
			log.Printf("Error registering whatsapp client: %v", err)
		}
	}

	eventsubManager := twitchLib.RegisterEventSub()

	fmt.Println("Starting server on port 4090")

	app := fiber.New()
	routes.RegisterRoutes(app)

	// Set up signal handling BEFORE starting the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := app.Listen(":4090"); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-c
	log.Println("Received shutdown signal, cleaning up...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown components gracefully
	if whatsappClient != nil {
		log.Println("Disconnecting WhatsApp client...")
		whatsappClient.Disconnect()
	}

	if eventsubManager != nil {
		log.Println("Closing EventSub manager...")
		eventsubManager.Close()
	}

	log.Println("Shutting down HTTP server...")
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Shutdown complete")
}
