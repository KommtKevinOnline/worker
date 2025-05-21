package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
)

func handleMessage(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.Sender.String() == "120363190873003716@newsletter" {
			fmt.Println(v.Message.GetConversation())
		}
	}
}

func Register(database *sql.DB) (*whatsmeow.Client, error) {
	container := sqlstore.NewWithDB(database, "postgres", nil)

	container.Upgrade(context.Background())

	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		fmt.Println("[WHATSAPP] Error getting first device:", err)
		return nil, err
	}

	fmt.Println("[WHATSAPP] Registering client...")

	client := whatsmeow.NewClient(deviceStore, nil)
	client.AddEventHandler(handleMessage)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				fmt.Println("[WHATSAPP] Scan the QR code to login")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("[WHATSAPP] Login event:", evt.Event)

				if evt.Event == "success" {
					//print newsletter channels
					newsletters, err := client.GetSubscribedNewsletters()
					if err != nil {
						fmt.Println("[WHATSAPP] Error getting subscribed newsletters:", err)
					}

					for _, newsletter := range newsletters {
						fmt.Println("[WHATSAPP] Subscribed newsletter:", newsletter.ThreadMeta.Name, newsletter.ID)
					}

					fmt.Println("[WHATSAPP] copy a channel id and set it as env variable: WHATSAPP_NEWSLETTER_CHANNEL_ID")
				}
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	return client, nil
}
