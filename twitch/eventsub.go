package twitch

import (
	"fmt"
	"os"

	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Adeithe/go-twitch/api"
	eventsub "github.com/joeyak/go-twitch-eventsub/v3"
	"github.com/samber/lo"
	"niki2k1.dev/m/postgres"
	"niki2k1.dev/m/queue"
)

func onStreamerOffline() {
	alreadyDownloaded, err := postgres.GetDownloadedVods()

	if err != nil {
		panic(err)
	}

	videos, err := GetLatestVideos()

	if err != nil {
		panic(err)
	}

	lo.ForEach(*videos, func(video api.Video, index int) {
		if lo.Contains(alreadyDownloaded, video.ID) {
			return
		}

		fmt.Printf("New Vod \"%s\" found.\n", video.ID)
		queue.AddToQueue(&video)
	})

	queue.Process()
}

func RegisterEventSub() *eventsub.Client {
	client := eventsub.NewClient()

	client.OnError(func(err error) {
		fmt.Printf("ERROR: %v\n", err)
	})

	client.OnWelcome(func(message eventsub.WelcomeMessage) {
		accessToken, _, _, err := postgres.GetTwitchLoginData()
		if err != nil {
			log.Printf("Failed to get twitch login data: %v", err)
			return
		}

		_, err = eventsub.SubscribeEvent(eventsub.SubscribeRequest{
			SessionID:   message.Payload.Session.ID,
			ClientID:    os.Getenv("TWITCH_CLIENT_ID"),
			AccessToken: accessToken,
			Event:       eventsub.SubStreamOffline,
			Condition: map[string]string{
				"broadcaster_user_id": os.Getenv("TWITCH_STREAMER_ID"),
			},
		})
		if err != nil {
			fmt.Printf("ERROR subscribing: %v\n", err)
			return
		}
	})

	client.OnEventStreamOffline(func(event eventsub.EventStreamOffline) {
		fmt.Printf("Received stream.offline event: %+v\n", event)
		onStreamerOffline()
	})

	if err := client.Connect(); err != nil {
		fmt.Printf("Could not connect client: %v\n", err)
	}

	return client
}

func SetupOauth() {
	accessToken, refreshToken, expiresIn, err := postgres.GetTwitchLoginData()
	if err != nil {
		log.Printf("Failed to get twitch login data: %v", err)
	}

	if accessToken != "" && refreshToken != "" && expiresIn != 0 {
		return
	}

	redirectURI := "http://localhost:8080/oauth/callback"
	scopes := "user:read:email" // Add more scopes as needed

	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")

	// 1. Start web server in a goroutine
	http.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		code := query.Get("code")

		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		// 4. Exchange code for tokens
		data := url.Values{}
		data.Set("client_id", clientID)
		data.Set("client_secret", clientSecret)
		data.Set("code", code)
		data.Set("grant_type", "authorization_code")
		data.Set("redirect_uri", redirectURI)

		fmt.Println(data.Encode())

		resp, err := http.Post(
			"https://id.twitch.tv/oauth2/token",
			"application/x-www-form-urlencoded",
			strings.NewReader(data.Encode()),
		)
		if err != nil {
			log.Printf("Token exchange failed: %v", err)
			http.Error(w, "Token exchange failed", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			log.Printf("Token exchange error: %s", string(body))
			http.Error(w, "Token exchange error", http.StatusInternalServerError)
			return
		}
		var tokenResp struct {
			AccessToken  string   `json:"access_token"`
			RefreshToken string   `json:"refresh_token"`
			ExpiresIn    int      `json:"expires_in"`
			Scope        []string `json:"scope"`
			TokenType    string   `json:"token_type"`
		}
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			log.Printf("Failed to parse token response: %v", err)
			http.Error(w, "Failed to parse token response", http.StatusInternalServerError)
			return
		}
		// 5. Save login data to DB (implement this function as needed)
		if err := postgres.SaveTwitchLoginData(tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn); err != nil {
			log.Printf("Failed to save login data: %v", err)
			http.Error(w, "Failed to save login data", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Twitch login successful! You can close this window."))
		log.Println("Twitch login successful and data saved.")

	})
	server := &http.Server{Addr: ":8080"}
	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	// 2. Print the login URL
	loginURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
	)
	fmt.Println("Please log in to Twitch by visiting this URL:")
	fmt.Println(loginURL)
	fmt.Println("Waiting for Twitch OAuth callback...")

	// 3. Wait for the answer (block main goroutine)
	select {}
}
