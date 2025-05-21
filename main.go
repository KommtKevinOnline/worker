package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"niki2k1.dev/m/postgres"
	twitchLib "niki2k1.dev/m/twitch"
	"niki2k1.dev/m/whatsapp"
)

func main() {
	godotenv.Load()

	database := postgres.GetConnection()

	twitchLib.SetupOauth()

	whatsappClient, err := whatsapp.Register(database)
	eventsubClient := twitchLib.RegisterEventSub()

	if err != nil {
		println("Error registering whatsapp client: ", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		whatsappClient.Disconnect()
		eventsubClient.Close()
		os.Exit(0)
	}()

}
