package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// Global map to store users (in a real app, you'd use a database)
var users = make(map[int64]*User)

// Load users from a JSON file
func loadUsers() {
	data, err := os.ReadFile("users.json")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading users file: %v", err)
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

	if err := os.WriteFile("users.json", data, 0644); err != nil {
		log.Printf("Error writing users file: %v", err)
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

func main() {
	// Load existing users
	loadUsers()

	// Set up Telegram bot
	bot, err := tgbotapi.NewBotAPI("7796841671:AAH9YeNYWzn5ChMAqal_DKYauUBe0nrFa84")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Comment out webhook setup for local testing
	// webhookURL := "https://dbotblock29.site/bot"
	// _, err = bot.Request(tgbotapi.NewSetWebhook(webhookURL))
	// if err != nil {
	//     log.Fatal(err)
	// }

	// Use long polling instead for local testing
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Start a goroutine to handle Telegram updates
	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
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
			default:
				msg.Text = "I don't understand that command. Try /start, /checkin, /share, or /balance."
			}

			if _, err := bot.Send(msg); err != nil {
				log.Println(err)
			}
		}
	}()

	// Comment out the webhook handler
	// http.HandleFunc("/bot", func(w http.ResponseWriter, r *http.Request) {
	// 	update, err := bot.HandleUpdate(r)
	// 	if err != nil {
	// 		log.Println(err)
	// 		return
	// 	}

	// 	if update.Message == nil {
	// 		return
	// 	}

	// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	// 	userID := update.Message.From.ID
	// 	username := update.Message.From.UserName

	// 	switch update.Message.Text {
	// 	case "/start":
	// 		msg.Text = "Welcome to Dots Rewards! 🎉\n\n" +
	// 			"Earn dots daily and exchange them for tokens later.\n\n" +
	// 			"Commands:\n" +
	// 			"/checkin - Get 10 dots daily\n" +
	// 			"/share - Get 20 dots for sharing\n" +
	// 			"/balance - Check your dots balance"
	// 	case "/checkin":
	// 		if canClaimDaily(userID) {
	// 			dots := awardDailyDots(userID, username)
	// 			msg.Text = fmt.Sprintf("✅ Daily check-in successful! You received 10 dots.\nYour balance: %d dots", dots)
	// 		} else {
	// 			msg.Text = "❌ You've already claimed your daily reward. Come back tomorrow!"
	// 		}
	// 	case "/share":
	// 		if canClaimShareReward(userID) {
	// 			dots := awardShareDots(userID, username)
	// 			msg.Text = fmt.Sprintf("✅ Thanks for sharing! You received 20 dots.\nYour balance: %d dots", dots)
	// 		} else {
	// 			msg.Text = "❌ You've already claimed your share reward today. Come back tomorrow!"
	// 		}
	// 	case "/balance":
	// 		dots := getUserDots(userID)
	// 		msg.Text = fmt.Sprintf("💰 Your current balance: %d dots", dots)
	// 	default:
	// 		msg.Text = "I don't understand that command. Try /start, /checkin, /share, or /balance."
	// 	}

	// 	if _, err := bot.Send(msg); err != nil {
	// 		log.Println(err)
	// 	}
	// })

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

	// Serve static files for the web interface
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Start the server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
