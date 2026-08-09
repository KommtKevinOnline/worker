package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"kommtkevinonline.de/models"
)

// Twitch rotates the refresh token on every refresh and invalidates the old
// one, so two concurrent refreshes can permanently lose the credentials.
var refreshMu sync.Mutex

func RefreshToken() {
	refreshMu.Lock()
	defer refreshMu.Unlock()

	token, err := models.TwitchToken{}.Get()
	if err != nil {
		log.Printf("Failed to get twitch login data: %v", err)
		return
	}

	// Refresh slightly early so an about-to-expire token is never handed out.
	expiryTime := token.CreatedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
	if time.Now().Add(time.Minute).Before(expiryTime) {
		return
	}

	fmt.Println("Token expired, refreshing...")

	data := url.Values{}
	data.Set("client_id", os.Getenv("TWITCH_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("TWITCH_CLIENT_SECRET"))
	data.Set("refresh_token", token.RefreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := http.Post(
		"https://id.twitch.tv/oauth2/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)

	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		return
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("Token exchange error: %s", string(body))
		return
	}

	if err := json.Unmarshal(body, &token); err != nil {
		log.Printf("Failed to parse token response: %v", err)
		return
	}

	if err := token.Save(); err != nil {
		// Losing the rotated refresh token here bricks auth until a manual
		// re-login, so make it loud.
		log.Printf("CRITICAL: failed to persist rotated twitch token: %v", err)
		return
	}
	fmt.Println("Token refreshed successfully")
}

func SetupOauth() {
	token, err := models.TwitchToken{}.Get()
	if err != nil {
		log.Printf("Failed to get twitch login data: %v", err)
	}

	if token.AccessToken != "" && token.RefreshToken != "" && token.ExpiresIn != 0 {
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

		token := models.TwitchToken{}

		if err := json.Unmarshal(body, &token); err != nil {
			log.Printf("Failed to parse token response: %v", err)
			http.Error(w, "Failed to parse token response", http.StatusInternalServerError)
			return
		}

		if err := token.Save(); err != nil {
			log.Printf("Failed to save twitch token: %v", err)
			http.Error(w, "Failed to save token", http.StatusInternalServerError)
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
