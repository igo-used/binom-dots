package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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
var dataFile = "users.json"
var lastSaveTime = time.Now()
var bot *tgbotapi.BotAPI

// Load users from the JSON file
func loadUsers() {
	// Pull latest changes from GitHub first
	pullFromGitHub()

	data, err := ioutil.ReadFile(dataFile)
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

// Save users to the JSON file and push to GitHub
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

	// Only push to GitHub every 5 minutes to avoid too many commits
	if time.Since(lastSaveTime).Minutes() >= 5 {
		pushToGitHub()
		lastSaveTime = time.Now()
	}
}

// Pull the latest data from GitHub
func pullFromGitHub() {
	cmd := exec.Command("git", "pull", "origin", "main")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error pulling from GitHub: %v - %s", err, output)
	} else {
		log.Printf("Successfully pulled from GitHub: %s", output)
	}
}

// Push changes to GitHub
func pushToGitHub() {
	// Add the users.json file
	addCmd := exec.Command("git", "add", dataFile)
	output, err := addCmd.CombinedOutput()
	if err != nil {
		log.Printf("Error adding file to git: %v - %s", err, output)
		return
	}

	// Commit the changes
	commitCmd := exec.Command("git", "commit", "-m", "Update user data")
	output, err = commitCmd.CombinedOutput()
	if err != nil {
		// If nothing to commit, that's fine
		if string(output) != "nothing to commit, working tree clean" {
			log.Printf("Error committing changes: %v - %s", err, output)
		}
		return
	}

	// Push to GitHub
	pushCmd := exec.Command("git", "push", "origin", "main")
	output, err = pushCmd.CombinedOutput()
	if err != nil {
		log.Printf("Error pushing to GitHub: %v - %s", err, output)
		return
	}

	log.Printf("Successfully pushed data to GitHub")
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

	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	// Start the server
	log.Println("Starting server on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
