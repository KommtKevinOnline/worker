package whatsapp

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"mime"
	"os"
	"text/tabwriter"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	"niki2k1.dev/m/transcriber"
)

func handleMessage(client *whatsmeow.Client, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		channelID := ensureChannelIdIsSet(client)
		if v.Info.Sender.String() == channelID {
			// If message contains text
			if v.Message.GetConversation() != "" {
				fmt.Println(v.Message.GetConversation())
			}

			// If message contains voice message
			if v.Message.AudioMessage != nil {
				data, err := client.Download(context.Background(), v.Message.AudioMessage)
				if err != nil {
					fmt.Println("[WHATSAPP] Error downloading audio message:", err)
				}
				exts, _ := mime.ExtensionsByType(v.Message.AudioMessage.GetMimetype())
				path := fmt.Sprintf("%s%s", v.Info.ID, exts[0])
				err = os.WriteFile(path, data, 0600)
				if err != nil {
					fmt.Println("[WHATSAPP] Error saving audio message:", err)
				}

				transcript, err := transcriber.Transcribe(bytes.NewReader(data))
				if err != nil {
					fmt.Println("[WHATSAPP] Error transcribing audio message:", err)
				}
				fmt.Println(transcript)
			}
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

	client := whatsmeow.NewClient(deviceStore, nil)
	client.AddEventHandler(func(evt interface{}) {
		handleMessage(client, evt)
	})

	fmt.Println("[WHATSAPP] Registered client")

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
					ensureChannelIdIsSet(client)
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

func ensureChannelIdIsSet(client *whatsmeow.Client) string {
	channelID := os.Getenv("WHATSAPP_NEWSLETTER_CHANNEL_ID")

	if channelID == "" {
		fmt.Println("[WHATSAPP] No WHATSAPP_NEWSLETTER_CHANNEL_ID env variable set, choose a channel:")

		listChannels(client)

		os.Exit(0)
	}

	return channelID
}

func listChannels(client *whatsmeow.Client) {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprintln(w, "ID\tName")

	channels, err := client.GetSubscribedNewsletters()
	if err != nil {
		fmt.Println("[WHATSAPP] Error getting subscribed newsletters:", err)
		return
	}

	for _, channel := range channels {
		fmt.Fprintf(w, "%s\t%s\n", channel.ID, channel.ThreadMeta.Name.Text)
	}

	w.Flush()
}
