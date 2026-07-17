package whatsapp

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	"kommtkevinonline.de/ai"
)

func handleMessage(client *whatsmeow.Client, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		channelID := ensureChannelIdIsSet(client)
		if v.Info.Sender.String() != channelID {
			return
		}

		var text string

		if conversation := v.Message.GetConversation(); conversation != "" {
			text = conversation
		} else if extended := v.Message.GetExtendedTextMessage().GetText(); extended != "" {
			text = extended
		} else if v.Message.AudioMessage != nil {
			data, err := client.Download(context.Background(), v.Message.AudioMessage)
			if err != nil {
				log.Printf("[WHATSAPP] Error downloading audio message: %v", err)
				return
			}

			transcript, err := ai.Transcribe(bytes.NewReader(data))
			if err != nil {
				log.Printf("[WHATSAPP] Error transcribing audio message: %v", err)
				return
			}

			text = transcript.Text
		}

		if text == "" {
			return
		}

		log.Printf("[WHATSAPP] Channel message: %s", text)

		predict(text, v.Info.ID, v.Info.Timestamp)
	}
}

func predict(text string, messageID string, sentAt time.Time) {
	loc, err := time.LoadLocation(os.Getenv("TZ"))
	if err != nil {
		loc, _ = time.LoadLocation("Europe/Berlin")
	}

	input := ai.Input{
		Text:          text,
		Source:        "whatsapp",
		ClipID:        messageID,
		ReferenceTime: sentAt,
	}

	outcome, err := ai.RunPrediction(context.Background(), ai.NewOpenAIClient(), ai.NewDBStore(), input, loc)
	if err != nil {
		log.Printf("[WHATSAPP] Error predicting from message: %v", err)
		return
	}

	log.Printf("[WHATSAPP] Message (model %s) saved %d predictions, cleared %d days", outcome.ModelUsed, len(outcome.Saved), len(outcome.ClearedDays))
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
			return nil, err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
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
			return nil, err
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
