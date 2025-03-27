package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// User represents a user in our system
type User struct {
	ID              int64     `json:"id"`
	Username        string    `json:"username"`
	LastCheckIn     time.Time `json:"last_check_in"`
	Dots            int       `json:"dots"`
	LastShareReward time.Time `json:"last_share_reward"`
}

// Global map to store users
var users = make(map[int64]*User)
var dataFile = "/data/users.json" // Using Render's persistent disk
var bot *tgbotapi.BotAPI

// Load users from the JSON file
func loadUsers() {
	// Create the /data directory if it doesn't exist
	os.MkdirAll("/data", 0755)

	// Check if the file exists
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		// Create an empty users file
		emptyUsers, _ := json.Marshal([]User{})
		ioutil.WriteFile(dataFile, emptyUsers, 0644)
		log.Printf("Created empty users file")
		return
	}

	data, err := ioutil.ReadFile(dataFile)
	if err != nil {
		log.Printf("Error reading users file: %v", err)
		return
	}

	var userList []*User
	if err := json.Unmarshal(data, &userList); err != nil {
		log.Printf("Error unmarshaling users: %v", err)
		return
	}

	for _, user := range userList {
		users[user.ID] = user
	}
	log.Printf("Loaded %d users", len(users))
}

// Save users to the JSON file
func saveUsers() {
	userList := make([]*User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	data, err := json.MarshalIndent(userList, "", "  ")
	if err != nil {
		log.Printf("Error marshaling users: %v", err)
		return
	}

	if err := ioutil.WriteFile(dataFile, data, 0644); err != nil {
		log.Printf("Error writing users file: %v", err)
		return
	}

	log.Printf("Saved %d users to disk", len(users))
}

// Check if a user can claim daily reward
func canClaimDaily(userID int64) bool {
	user, exists := users[userID]
	if !exists {
		return true
	}

	// Check if it's past 01:00 GMT+1 today
	now := time.Now().UTC().Add(time.Hour) // GMT+1
	resetTime := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, time.UTC)

	// If current time is before 01:00, use yesterday's reset time
	if now.Hour() < 1 {
		yesterday := now.AddDate(0, 0, -1)
		resetTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 1, 0, 0, 0, time.UTC)
	}

	// User can claim if their last check-in was before today's reset time
	return user.LastCheckIn.Before(resetTime)
}

// Check if a user can claim share reward
func canClaimShareReward(userID int64) bool {
	user, exists := users[userID]
	if !exists {
		return true
	}

	// Check if it's past 01:00 GMT+1 today
	now := time.Now().UTC().Add(time.Hour) // GMT+1
	resetTime := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, time.UTC)

	// If current time is before 01:00, use yesterday's reset time
	if now.Hour() < 1 {
		yesterday := now.AddDate(0, 0, -1)
		resetTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 1, 0, 0, 0, time.UTC)
	}

	// User can claim if their last share reward was before today's reset time
	return user.LastShareReward.Before(resetTime)
}

// Award daily dots to a user
func awardDailyDots(userID int64, username string) int {
	user, exists := users[userID]
	if !exists {
		user = &User{
			ID:       userID,
			Username: username,
			Dots:     0,
		}
		users[userID] = user
	}

	user.Dots += 10
	user.LastCheckIn = time.Now()
	saveUsers()
	return user.Dots
}

// Award share dots to a user
func awardShareDots(userID int64, username string) int {
	user, exists := users[userID]
	if !exists {
		user = &User{
			ID:       userID,
			Username: username,
			Dots:     0,
		}
		users[userID] = user
	}

	user.Dots += 20
	user.LastShareReward = time.Now()
	saveUsers()
	return user.Dots
}

// Get user dots
func getUserDots(userID int64) int {
	user, exists := users[userID]
	if !exists {
		return 0
	}
	return user.Dots
}

// Handle Telegram bot commands
func handleTelegramCommand(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	userID := update.Message.From.ID
	username := update.Message.From.UserName

	switch update.Message.Text {
	case "/start":
		msg.Text = "Welcome to Dots Rewards! 🎉\n\n" +
			"Earn dots daily and exchange them for tokens later.\n\n" +
			"Commands:\n" +
			"/checkin - Get 10 dots daily\n" +
			"/share - Get 20 dots for sharing\n" +
			"/balance - Check your dots balance"
	case "/checkin":
		if canClaimDaily(userID) {
			dots := awardDailyDots(userID, username)
			msg.Text = fmt.Sprintf("✅ Daily check-in successful! You received 10 dots.\nYour balance: %d dots", dots)
		} else {
			msg.Text = "❌ You've already claimed your daily reward. Come back after 01:00 GMT+1!"
		}
	case "/share":
		if canClaimShareReward(userID) {
			dots := awardShareDots(userID, username)
			msg.Text = fmt.Sprintf("✅ Thanks for sharing! You received 20 dots.\nYour balance: %d dots", dots)
		} else {
			msg.Text = "❌ You've already claimed your share reward today. Come back after 01:00 GMT+1!"
		}
	case "/balance":
		dots := getUserDots(userID)
		msg.Text = fmt.Sprintf("💰 Your current balance: %d dots", dots)
	default:
		msg.Text = "I don't understand that command. Try /start, /checkin, /share, or /balance."
	}

	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}
}

func main() {
	// Load existing users
	loadUsers()

	// Set up Telegram bot
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Set webhook
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "https://binom-dots.onrender.com/bot"
	}

	// First, delete any existing webhook
	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		log.Printf("Error deleting webhook: %v", err)
	}

	// Parse the webhook URL and set it
	webhookURLParsed, err := url.Parse(webhookURL)
	if err != nil {
		log.Printf("Error parsing webhook URL: %v", err)
	} else {
		_, err = bot.Request(tgbotapi.WebhookConfig{
			URL: webhookURLParsed,
		})
		if err != nil {
			log.Printf("Error setting webhook: %v", err)
		}
	}

	// Set up webhook handler
	http.HandleFunc("/bot", func(w http.ResponseWriter, r *http.Request) {
		update, err := bot.HandleUpdate(r)
		if err != nil {
			log.Printf("Error handling update: %v", err)
			return
		}

		if update != nil {
			handleTelegramCommand(*update)
		}
	})

	// Add a health check endpoint for Render
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API endpoints for the web interface
	http.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		userIDStr := r.URL.Query().Get("id")
		if userIDStr == "" {
			http.Error(w, "Missing user ID", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		user, exists := users[userID]
		if !exists {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	// API endpoint for daily check-in
	http.HandleFunc("/api/checkin", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		userIDStr := r.URL.Query().Get("id")
		if userIDStr == "" {
			http.Error(w, "Missing user ID", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		user, exists := users[userID]
		if !exists {
			// Create new user
			user = &User{
				ID:       userID,
				Username: "",
				Dots:     0,
			}
			users[userID] = user
		}

		if canClaimDaily(userID) {
			dots := awardDailyDots(userID, user.Username)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"dots":    dots,
				"message": "Daily check-in successful",
			})
		} else {
			http.Error(w, "Already claimed today", http.StatusBadRequest)
		}
	})

	// API endpoint for sharing
	http.HandleFunc("/api/share", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		userIDStr := r.URL.Query().Get("id")
		if userIDStr == "" {
			http.Error(w, "Missing user ID", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		user, exists := users[userID]
		if !exists {
			// Create new user
			user = &User{
				ID:       userID,
				Username: "",
				Dots:     0,
			}
			users[userID] = user
		}

		if canClaimShareReward(userID) {
			dots := awardShareDots(userID, user.Username)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"dots":    dots,
				"message": "Share reward claimed successfully",
			})
		} else {
			http.Error(w, "Already claimed today", http.StatusBadRequest)
		}
	})

	// Serve static files for the web interface
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // Default port if not specified
	}

	// Start the server
	log.Println("Starting server on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
