package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
		log.Printf("Git commit output: %s", output)
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

// Get a user from the local map
func getUser(userID int64) (*User, error) {
	user, exists := users[userID]
	if !exists {
		return nil, nil
	}
	return user, nil
}

// Save a user to the local map and trigger JSON save
func saveUser(user *User) error {
	// Log the user data being saved
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshaling user data: %v", err)
		return err
	}
	log.Printf("Saving user data: %s", string(userData))

	// Save to the map
	users[user.ID] = user

	// Save to JSON file
	saveUsers()

	return nil
}

// Check if a user can claim daily reward
func canClaimDaily(userID int64) bool {
	user, err := getUser(userID)
	if err != nil || user == nil {
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
	user, err := getUser(userID)
	if err != nil || user == nil {
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
func awardDailyDots(userID int64, username string) (int, error) {
	user, err := getUser(userID)
	if err != nil {
		return 0, err
	}

	if user == nil {
		// Create new user
		user = &User{
			ID:       userID,
			Username: username,
			Dots:     0,
		}
	}

	user.Dots += 10
	user.LastCheckIn = time.Now()

	if err := saveUser(user); err != nil {
		return 0, err
	}

	return user.Dots, nil
}

// Award share dots to a user
func awardShareDots(userID int64, username string) (int, error) {
	user, err := getUser(userID)
	if err != nil {
		return 0, err
	}

	if user == nil {
		// Create new user
		user = &User{
			ID:       userID,
			Username: username,
			Dots:     0,
		}
	}

	user.Dots += 20
	user.LastShareReward = time.Now()

	if err := saveUser(user); err != nil {
		return 0, err
	}

	return user.Dots, nil
}

// Get user dots
func getUserDots(userID int64) (int, error) {
	user, err := getUser(userID)
	if err != nil {
		return 0, err
	}

	if user == nil {
		return 0, nil
	}

	return user.Dots, nil
}

func main() {
	// Load existing users
	loadUsers()

	// Get bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		botToken = "7796841671:AAH9YeNYWzn5ChMAqal_DKYauUBe0nrFa84" // Fallback to hardcoded token
		log.Println("Using hardcoded bot token. Consider setting TELEGRAM_BOT_TOKEN environment variable.")
	}

	// Set up Telegram bot
	bot, err := tgbotapi.NewBotAPI(botToken)
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

	// Add webhook handler
	http.HandleFunc("/bot", func(w http.ResponseWriter, r *http.Request) {
		update, err := bot.HandleUpdate(r)
		if err != nil {
			log.Println(err)
			return
		}

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
				dots, err := awardDailyDots(userID, username)
				if err != nil {
					log.Printf("Error awarding daily dots: %v", err)
					msg.Text = "❌ Error claiming daily reward. Please try again later."
				} else {
					msg.Text = fmt.Sprintf("✅ Daily check-in successful! You received 10 dots.\nYour balance: %d dots", dots)
				}
			} else {
				msg.Text = "❌ You've already claimed your daily reward. Come back after 01:00 GMT+1!"
			}
		case "/share":
			if canClaimShareReward(userID) {
				dots, err := awardShareDots(userID, username)
				if err != nil {
					log.Printf("Error awarding share dots: %v", err)
					msg.Text = "❌ Error claiming share reward. Please try again later."
				} else {
					msg.Text = fmt.Sprintf("✅ Thanks for sharing! You received 20 dots.\nYour balance: %d dots", dots)
				}
			} else {
				msg.Text = "❌ You've already claimed your share reward today. Come back after 01:00 GMT+1!"
			}
		case "/balance":
			dots, err := getUserDots(userID)
			if err != nil {
				log.Printf("Error getting user dots: %v", err)
				msg.Text = "❌ Error checking balance. Please try again later."
			} else {
				msg.Text = fmt.Sprintf("💰 Your current balance: %d dots", dots)
			}
		default:
			msg.Text = "I don't understand that command. Try /start, /checkin, /share, or /balance."
		}

		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
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

		user, err := getUser(userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "Error getting user", http.StatusInternalServerError)
			return
		}

		if user == nil {
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

		user, err := getUser(userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "Error getting user", http.StatusInternalServerError)
			return
		}

		if user == nil {
			// Create new user
			user = &User{
				ID:       userID,
				Username: "",
				Dots:     0,
			}
		}

		if canClaimDaily(userID) {
			dots, err := awardDailyDots(userID, user.Username)
			if err != nil {
				log.Printf("Error awarding daily dots: %v", err)
				http.Error(w, "Error awarding daily dots", http.StatusInternalServerError)
				return
			}

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

		user, err := getUser(userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "Error getting user", http.StatusInternalServerError)
			return
		}

		if user == nil {
			// Create new user
			user = &User{
				ID:       userID,
				Username: "",
				Dots:     0,
			}
		}

		if canClaimShareReward(userID) {
			dots, err := awardShareDots(userID, user.Username)
			if err != nil {
				log.Printf("Error awarding share dots: %v", err)
				http.Error(w, "Error awarding share dots", http.StatusInternalServerError)
				return
			}

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

	// Add a health check endpoint for Render
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
		port = "10000" // Match your Render PORT setting
	}
	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
