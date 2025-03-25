document.addEventListener("DOMContentLoaded", () => {
  const checkinBtn = document.getElementById("checkin-btn")
  const shareBtn = document.getElementById("share-btn")
  const telegramBtn = document.getElementById("telegram-btn")
  const dotsCount = document.getElementById("dots-count")
  const shareOptions = document.getElementById("share-options")
  const shareButtons = document.querySelectorAll(".share-btn")

  let userId = localStorage.getItem("userId")
  let userDots = 0

  // Add a manual connect button
  const manualConnectBtn = document.createElement("button")
  manualConnectBtn.className = "btn"
  manualConnectBtn.textContent = "Connect with Telegram ID"
  manualConnectBtn.style.marginTop = "10px"
  telegramBtn.parentNode.appendChild(manualConnectBtn)

  // Function to prompt for Telegram ID
  function promptForTelegramId() {
    const telegramId = prompt(
      "Please enter your Telegram ID\n\n" +
      "To find your ID:\n" +
      "1. Open Telegram\n" +
      "2. Message our bot @BinomChain_bot\n" +
      "3. Send the command /id\n" +
      "4. Copy the number shown"
    )
    
    if (telegramId && !isNaN(telegramId)) {
      userId = telegramId
      localStorage.setItem("userId", userId)
      
      // Try to fetch user, create if not found
      fetch(`https://binom-dots.onrender.com/api/user?id=${userId}`)
        .then(response => {
          if (!response.ok) {
            if (response.status === 404) {
              // User doesn't exist, create them
              console.log("User not found, creating new user...")
              return fetch(`https://binom-dots.onrender.com/api/user/create?id=${userId}`, {
                method: "POST"
              })
            }
            throw new Error("Failed to connect")
          }
          return response.json()
        })
        .then(data => {
          console.log("Connection successful:", data)
          checkUserConnection()
          alert("Connected successfully! Your Telegram ID: " + userId)
        })
        .catch(error => {
          console.error("Connection error:", error)
          alert("Error connecting: " + error.message)
          localStorage.removeItem("userId")
        })
    } else {
      alert("Invalid Telegram ID. Please try again.")
    }
  }

  // Add event listener for manual connect button
  manualConnectBtn.addEventListener("click", promptForTelegramId)

  // Check if user is connected
  function checkUserConnection() {
    if (userId) {
      // User is connected, fetch their data
      fetchUserData()
      telegramBtn.textContent = "Connected with Telegram"
      telegramBtn.disabled = true
      manualConnectBtn.style.display = "none"
      checkinBtn.disabled = false
      shareBtn.disabled = false
    } else {
      // User is not connected
      checkinBtn.disabled = true
      shareBtn.disabled = true
      manualConnectBtn.style.display = "block"
    }
  }

  // Check if the time has passed 01:00 GMT+1
  function hasPassedResetTime(lastTime) {
    if (!lastTime) return true;
    
    const now = new Date();
    const lastDate = new Date(lastTime);
    
    // Create reset time for today at 01:00 GMT+1
    const resetTime = new Date(now);
    resetTime.setHours(1, 0, 0, 0); // 01:00:00.000
    resetTime.setMinutes(resetTime.getMinutes() + 60); // Add 1 hour for GMT+1
    
    // If now is before today's reset time, use yesterday's reset time
    if (now < resetTime) {
      resetTime.setDate(resetTime.getDate() - 1);
    }
    
    // Check if the last claim was before the reset time
    return lastDate < resetTime;
  }

  // Fetch user data from the API
  function fetchUserData() {
    if (!userId) return

    console.log("Fetching data for user ID:", userId)

    fetch(`https://binom-dots.onrender.com/api/user?id=${userId}`)
      .then((response) => {
        if (!response.ok) {
          if (response.status === 404) {
            // Try to create the user
            return fetch(`https://binom-dots.onrender.com/api/user/create?id=${userId}`, {
              method: "POST"
            })
          }
          throw new Error("User not found")
        }
        return response.json()
      })
      .then((data) => {
        console.log("User data received:", data)
        userDots = data.dots
        dotsCount.textContent = userDots

        // Check if daily rewards are available using the new reset time logic
        const checkInAvailable = hasPassedResetTime(data.last_check_in);
        const shareAvailable = hasPassedResetTime(data.last_share_reward);

        checkinBtn.disabled = !checkInAvailable
        shareBtn.disabled = !shareAvailable

        if (!checkInAvailable) {
          checkinBtn.textContent = "Claimed Today"
        } else {
          checkinBtn.textContent = "Claim"
        }

        if (!shareAvailable) {
          shareBtn.textContent = "Shared Today"
        } else {
          shareBtn.textContent = "Share"
        }
      })
      .catch((error) => {
        console.error("Error fetching user data:", error)
        // If user not found and creation failed, clear localStorage
        if (error.message === "User not found") {
          alert("User not found. Please connect with your Telegram ID again.")
          localStorage.removeItem("userId")
          checkUserConnection()
        }
      })
  }

  // Connect with Telegram
  telegramBtn.addEventListener("click", function() {
    console.log("Telegram button clicked") // Debug log
    
    // Open Telegram bot in a new window - CORRECTED BOT NAME
    window.open("https://t.me/BinomChain_bot", "_blank")

    // Show instructions to the user
    setTimeout(() => {
      alert(
        "1. Send /start to the bot\n" +
        "2. Send /id to get your Telegram ID\n" +
        "3. Come back here and click 'Connect with Telegram ID'\n" +
        "4. Enter your Telegram ID from the bot"
      )
    }, 500)
  })

  // Claim daily check-in reward
  checkinBtn.addEventListener("click", () => {
    if (!userId) return

    console.log("Claiming daily reward for user:", userId)

    // Make an actual API call to the backend
    fetch(`https://binom-dots.onrender.com/api/checkin?id=${userId}`, {
      method: "POST",
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to check in")
        }
        return response.json()
      })
      .then((data) => {
        console.log("Check-in successful:", data)
        userDots = data.dots
        dotsCount.textContent = userDots
        checkinBtn.disabled = true
        checkinBtn.textContent = "Claimed Today"
        alert("You claimed 10 dots for daily check-in!")
      })
      .catch((error) => {
        console.error("Error claiming daily reward:", error)
        alert("Failed to claim reward. Please try again later.")
      })
  })

  // Show share options
  shareBtn.addEventListener("click", () => {
    if (!userId) return

    console.log("Opening share options")

    // Show share options
    shareOptions.classList.add("active")

    // Scroll to share options
    shareOptions.scrollIntoView({ behavior: "smooth" })
  })

  // Handle share button clicks
  shareButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const platform = button.getAttribute("data-platform")
      let shareUrl = ""
      const shareText = "I'm collecting Binom Dots from Binomena Blockchain!"

      console.log("Sharing to platform:", platform)

      // Create share URLs for different platforms
      switch (platform) {
        case "instagram":
          // For Instagram, we can't directly share, but we can copy the text
          navigator.clipboard
            .writeText(shareText)
            .then(() => {
              alert("Text copied! Open Instagram and paste to share.")
              window.open("https://www.instagram.com/", "_blank")
            })
            .catch((err) => {
              console.error("Could not copy text: ", err)
              // Fallback
              window.open("https://www.instagram.com/", "_blank")
            })
          break

        case "telegram":
          shareUrl = `https://t.me/share/url?url=${encodeURIComponent("https://dbotblock29.site")}&text=${encodeURIComponent(shareText)}`
          window.open(shareUrl, "_blank")
          break

        case "twitter":
          shareUrl = `https://twitter.com/intent/tweet?text=${encodeURIComponent(shareText)}`
          window.open(shareUrl, "_blank")
          break

        case "whatsapp":
          shareUrl = `https://wa.me/?text=${encodeURIComponent(shareText)}`
          window.open(shareUrl, "_blank")
          break
      }

      // Make an actual API call to the backend
      fetch(`https://binom-dots.onrender.com/api/share?id=${userId}`, {
        method: "POST",
      })
        .then((response) => {
          if (!response.ok) {
            throw new Error("Failed to claim share reward")
          }
          return response.json()
        })
        .then((data) => {
          console.log("Share reward claimed:", data)
          userDots = data.dots
          dotsCount.textContent = userDots
          shareBtn.disabled = true
          shareBtn.textContent = "Shared Today"
          shareOptions.classList.remove("active")
          alert("Thanks for sharing! You earned 20 dots!")
        })
        .catch((error) => {
          console.error("Error claiming share reward:", error)
          alert("Failed to claim share reward. Please try again later.")
        })
    })
  })

  // Add animation effects to dots
  function animateDots() {
    const dots = document.querySelectorAll(".hero-image circle")
    dots.forEach((dot) => {
      const randomDelay = Math.random() * 5
      const randomDuration = 3 + Math.random() * 2

      dot.style.animation = `pulse ${randomDuration}s ease-in-out ${randomDelay}s infinite`
    })
  }

  // Initialize
  checkUserConnection()
  animateDots()
})