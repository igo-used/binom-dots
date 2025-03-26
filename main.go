package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// Supabase configuration
const (
	supabaseURL = "https://wzzaxbfdecshddqjfwxs.supabase.co"
	supabaseKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Ind6emF4YmZkZWNzaGRkcWpmd3hzIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDI5NjgwMTEsImV4cCI6MjA1ODU0NDAxMX0.FVhxImKL_DKaP5YAFx7ol9LsqtRSBFI0mKYluBh_6qM"
)

// Get a user from Supabase
func getUser(userID int64) (*User, error) {
	// Prepare the request
	url := fmt.Sprintf("%s/rest/v1/users?id=eq.%d", supabaseURL, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Add("apikey", supabaseKey)
	req.Header.Add("Authorization", "Bearer "+supabaseKey)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var users []*User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, err
	}

	// Check if user exists
	if len(users) == 0 {
		return nil, nil
	}

	return users[0], nil
}

// Create or update a user in Supabase
func saveUser(user *User) error {
	// Prepare the request
	url := fmt.Sprintf("%s/rest/v1/users", supabaseURL)
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(userData))
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Add("apikey", supabaseKey)
	req.Header.Add("Authorization", "Bearer "+supabaseKey)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Prefer", "resolution=merge-duplicates")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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
	// Get bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	// Set up Telegram bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// IMPORTANT: Delete any existing webhook first
	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		log.Printf("Error deleting webhook: %v", err)
	}

	// Use long polling for updates
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
		}
	}()

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
		port = "8080"
	}
	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
