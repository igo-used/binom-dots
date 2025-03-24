package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// File paths
var dataDir = "."
var usersFilePath = filepath.Join(dataDir, "users.json")

// Initialize data directory
func init() {
	// Check if we're on Render with a disk mount
	if _, err := os.Stat("/data"); err == nil {
		dataDir = "/data"
		usersFilePath = filepath.Join(dataDir, "users.json")
		log.Println("Using persistent data directory:", dataDir)
	} else {
		log.Println("Using local directory for data")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
	}
}

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

// Load users from a JSON file
func loadUsers() {
	log.Printf("Loading users from %s", usersFilePath)
	data, err := os.ReadFile(usersFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading users file: %v", err)
		} else {
			log.Printf("Users file does not exist yet, will create on first save")
		}
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

// Save users to a JSON file
func saveUsers() {
	userList := make([]*User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	data, err := json.Marshal(userList)
	if err != nil {
		log.Printf("Error marshaling users: %v", err)
		return
	}

	log.Printf("Saving %d users to %s", len(userList), usersFilePath)
	if err := os.WriteFile(usersFilePath, data, 0644); err != nil {
		log.Printf("Error writing users file: %v", err)
	} else {
		log.Printf("Successfully saved users to %s", usersFilePath)
	}
}

// Check if a user can claim daily reward
func canClaimDaily(userID int64) bool {
	user, exists := users[userID]
	if !exists {
		return true
	}

	now := time.Now()
	return now.Sub(user.LastCheckIn).Hours() >= 24
}

// Check if a user can claim share reward
func canClaimShareReward(userID int64) bool {
	user, exists := users[userID]
	if !exists {
		return true
	}

	now := time.Now()
	return now.Sub(user.LastShareReward).Hours() >= 24
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

// Process Telegram message
func processTelegramMessage(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
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
			"/balance - Check your dots balance\n" +
			"/id - Get your Telegram ID for the website"
	case "/checkin":
		if canClaimDaily(userID) {
			dots := awardDailyDots(userID, username)
			msg.Text = fmt.Sprintf("✅ Daily check-in successful! You received 10 dots.\nYour balance: %d dots", dots)
		} else {
			msg.Text = "❌ You've already claimed your daily reward. Come back tomorrow!"
		}
	case "/share":
		if canClaimShareReward(userID) {
			dots := awardShareDots(userID, username)
			msg.Text = fmt.Sprintf("✅ Thanks for sharing! You received 20 dots.\nYour balance: %d dots", dots)
		} else {
			msg.Text = "❌ You've already claimed your share reward today. Come back tomorrow!"
		}
	case "/balance":
		dots := getUserDots(userID)
		msg.Text = fmt.Sprintf("💰 Your current balance: %d dots", dots)
	case "/id":
		msg.Text = fmt.Sprintf("Your Telegram ID is: %d\n\nUse this ID to connect on our website: https://dbotblock29.site", userID)
	case "/help":
		msg.Text = "📚 *Binom Dots Bot Commands*\n\n" +
			"*/start* - Welcome message and introduction\n" +
			"*/checkin* - Claim your daily 10 dots reward\n" +
			"*/share* - Get 20 dots for sharing (once per day)\n" +
			"*/balance* - Check your current dots balance\n" +
			"*/id* - Get your Telegram ID for the website\n" +
			"*/help* - Show this help message"
		msg.ParseMode = "Markdown"
	case "/info":
		msg.Text = "ℹ️ *About Binom Dots*\n\n" +
			"Binom Dots is a rewards program by Binomena Blockchain.\n\n" +
			"• Collect dots daily by checking in and sharing\n" +
			"• 1000 dots = 1 token when our blockchain launches\n" +
			"• Visit our website: https://dbotblock29.site\n\n" +
			"Powered by ADA Neural technology"
		msg.ParseMode = "Markdown"
	default:
		msg.Text = "I don't understand that command. Try /start, /checkin, /share, /balance, or /help."
	}

	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Load existing users
	loadUsers()

	// Get bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	// Initialize bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Determine if we're in production or development
	isProduction := os.Getenv("ENVIRONMENT") == "production"

	if isProduction {
		// For production, we'll just use the webhook handler without trying to set it
		// The webhook should be set manually in the Telegram API
		log.Printf("Running in production mode with webhook handler")

		// Add webhook handler
		http.HandleFunc("/bot", func(w http.ResponseWriter, r *http.Request) {
			update, err := bot.HandleUpdate(r)
			if err != nil {
				log.Println(err)
				return
			}

			processTelegramMessage(update, bot)
		})

		// Log instructions for setting up the webhook manually
		log.Printf("IMPORTANT: Set your webhook manually by visiting:")
		log.Printf("https://api.telegram.org/bot%s/setWebhook?url=https://binom-dots.onrender.com/bot", botToken)
	} else {
		// Development: Use long polling with random offset to avoid conflicts
		log.Printf("Running in development mode with long polling")

		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		// Add a random offset to avoid conflicts with other instances
		// Convert int64 to int for the Offset field
		randomOffset := int(rand.Int63n(100) + 1)
		u.Offset = randomOffset

		updates := bot.GetUpdatesChan(u)

		// Start a goroutine to handle Telegram updates
		go func() {
			for update := range updates {
				processTelegramMessage(&update, bot)
			}
		}()
	}

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

	// API endpoint for user creation
	http.HandleFunc("/api/user/create", func(w http.ResponseWriter, r *http.Request) {
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

		// Create a new user if they don't exist
		user, exists := users[userID]
		if !exists {
			user = &User{
				ID:       userID,
				Username: "web_user", // Default username for web users
				Dots:     0,
			}
			users[userID] = user
			saveUsers() // Save to persistent storage
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
			http.Error(w, "User not found", http.StatusNotFound)
			return
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
			http.Error(w, "User not found", http.StatusNotFound)
			return
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

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files for the web interface
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
