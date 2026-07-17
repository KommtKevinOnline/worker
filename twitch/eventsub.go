package twitch

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Adeithe/go-twitch/api"
	eventsub "github.com/joeyak/go-twitch-eventsub/v3"
	"github.com/samber/lo"
	"kommtkevinonline.de/database"
	"kommtkevinonline.de/models"
	"kommtkevinonline.de/queue"
)

const (
	// Twitch sends keepalive every ~10 seconds, we allow some buffer
	keepaliveTimeout = 30 * time.Second
	// Maximum backoff time for reconnection attempts
	maxBackoff = 5 * time.Minute
	// Initial backoff time
	initialBackoff = 1 * time.Second
)

// EventSubManager manages the EventSub connection with automatic reconnection
type EventSubManager struct {
	client          *eventsub.Client
	mu              sync.Mutex
	lastMessage     time.Time
	stopChan        chan struct{}
	isConnected     bool
	backoff         time.Duration
	keepaliveTicker *time.Ticker
}

// NewEventSubManager creates a new EventSub manager
func NewEventSubManager() *EventSubManager {
	return &EventSubManager{
		stopChan: make(chan struct{}),
		backoff:  initialBackoff,
	}
}

func onStreamerOnline(event eventsub.EventStreamOnline) {
	stream := models.Stream{
		ID:        event.Id,
		StartedAt: event.StartedAt,
	}

	// Title and category are not part of the event payload; best effort only
	if current, err := GetCurrentStream(); err != nil {
		log.Printf("Error fetching current stream info: %v", err)
	} else if current != nil {
		stream.Title = current.Title
		stream.Category = current.GameName
	}

	if err := stream.Save(); err != nil {
		log.Printf("Error saving stream start: %v", err)
	}
}

func onStreamerOffline() {
	if err := models.MarkLatestStreamEnded(time.Now()); err != nil {
		log.Printf("Error marking stream as ended: %v", err)
	}

	alreadyDownloaded, err := database.GetDownloadedVods()

	if err != nil {
		log.Printf("Error getting downloaded vods: %v", err)
		return
	}

	videos, err := GetLatestVideos(20)

	if err != nil {
		log.Printf("Error getting latest videos: %v", err)
		return
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

// subscribeToEvents subscribes to the stream.offline event
func (m *EventSubManager) subscribeToEvents(sessionID string) error {
	// Refresh token before subscribing to ensure it's valid
	RefreshToken()

	token, err := models.TwitchToken{}.Get()
	if err != nil {
		return fmt.Errorf("failed to get twitch token: %w", err)
	}

	for _, eventType := range []eventsub.EventSubscription{eventsub.SubStreamOffline, eventsub.SubStreamOnline} {
		_, err = eventsub.SubscribeEvent(eventsub.SubscribeRequest{
			SessionID:   sessionID,
			ClientID:    os.Getenv("TWITCH_CLIENT_ID"),
			AccessToken: token.AccessToken,
			Event:       eventType,
			Condition: map[string]string{
				"broadcaster_user_id": os.Getenv("TWITCH_STREAMER_ID"),
			},
		})

		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", eventType, err)
		}

		log.Printf("Successfully subscribed to %s event", eventType)
	}

	return nil
}

// updateLastMessage updates the last message timestamp
func (m *EventSubManager) updateLastMessage() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastMessage = time.Now()
}

// startKeepaliveMonitor monitors for keepalive timeouts
func (m *EventSubManager) startKeepaliveMonitor() {
	m.keepaliveTicker = time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-m.stopChan:
				m.keepaliveTicker.Stop()
				return
			case <-m.keepaliveTicker.C:
				m.mu.Lock()
				lastMsg := m.lastMessage
				connected := m.isConnected
				m.mu.Unlock()

				if connected && time.Since(lastMsg) > keepaliveTimeout {
					log.Printf("Keepalive timeout detected (last message: %v ago), reconnecting...", time.Since(lastMsg))
					m.reconnect()
				}
			}
		}
	}()
}

// connect establishes the EventSub connection
func (m *EventSubManager) connect() {
	m.mu.Lock()
	if m.client != nil {
		m.client.Close()
	}

	m.client = eventsub.NewClient()
	m.isConnected = false
	m.mu.Unlock()

	// Handle errors
	m.client.OnError(func(err error) {
		log.Printf("EventSub error: %v", err)
		m.mu.Lock()
		m.isConnected = false
		m.mu.Unlock()
		// Schedule reconnection
		go m.scheduleReconnect()
	})

	// Handle welcome message - subscribe to events
	m.client.OnWelcome(func(message eventsub.WelcomeMessage) {
		log.Printf("EventSub connected, session ID: %s", message.Payload.Session.ID)
		m.updateLastMessage()

		m.mu.Lock()
		m.isConnected = true
		m.backoff = initialBackoff // Reset backoff on successful connection
		m.mu.Unlock()

		if err := m.subscribeToEvents(message.Payload.Session.ID); err != nil {
			log.Printf("Failed to subscribe: %v", err)
		}
	})

	// Handle keepalive messages
	m.client.OnKeepAlive(func(message eventsub.KeepAliveMessage) {
		m.updateLastMessage()
	})

	// Handle session reconnect - Twitch wants us to move to a new server
	m.client.OnReconnect(func(message eventsub.ReconnectMessage) {
		log.Printf("Received reconnect request from Twitch (new URL: %s)", message.Payload.Session.ReconnectUrl)
		m.updateLastMessage()

		// Close current connection and reconnect
		// This will create a new session with Twitch
		go m.reconnect()
	})

	// Handle stream online event - record the actual start for accuracy stats
	m.client.OnEventStreamOnline(func(event eventsub.EventStreamOnline) {
		log.Printf("Received stream.online event: broadcaster=%s stream=%s", event.BroadcasterUserName, event.Id)
		m.updateLastMessage()
		onStreamerOnline(event)
	})

	// Handle stream offline event
	m.client.OnEventStreamOffline(func(event eventsub.EventStreamOffline) {
		log.Printf("Received stream.offline event: broadcaster=%s", event.BroadcasterUserName)
		m.updateLastMessage()
		onStreamerOffline()
	})

	// Handle notification (updates last message time for any notification)
	m.client.OnNotification(func(message eventsub.NotificationMessage) {
		m.updateLastMessage()
	})

	// Connect
	go func() {
		if err := m.client.Connect(); err != nil {
			log.Printf("EventSub connection failed: %v", err)
			m.mu.Lock()
			m.isConnected = false
			m.mu.Unlock()
			go m.scheduleReconnect()
		}
	}()
}

// scheduleReconnect schedules a reconnection with exponential backoff
func (m *EventSubManager) scheduleReconnect() {
	m.mu.Lock()
	backoff := m.backoff
	// Increase backoff for next attempt (exponential backoff)
	m.backoff = m.backoff * 2
	if m.backoff > maxBackoff {
		m.backoff = maxBackoff
	}
	m.mu.Unlock()

	log.Printf("Scheduling reconnection in %v", backoff)

	select {
	case <-m.stopChan:
		return
	case <-time.After(backoff):
		m.reconnect()
	}
}

// reconnect closes the current connection and establishes a new one
func (m *EventSubManager) reconnect() {
	log.Printf("Reconnecting to EventSub...")

	// Refresh token before reconnecting
	RefreshToken()

	m.connect()
}

// Close closes the EventSub connection and stops monitoring
func (m *EventSubManager) Close() {
	log.Printf("Closing EventSub manager...")
	close(m.stopChan)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
	m.isConnected = false
}

// RegisterEventSub creates and starts the EventSub manager
func RegisterEventSub() *EventSubManager {
	// Ensure we have a valid token before connecting
	RefreshToken()

	manager := NewEventSubManager()
	manager.updateLastMessage() // Initialize last message time
	manager.startKeepaliveMonitor()
	manager.connect()

	return manager
}
