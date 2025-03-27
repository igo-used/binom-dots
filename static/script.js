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
    const telegramId = prompt("Please enter your Telegram user ID:", "")
    if (telegramId && !isNaN(telegramId)) {
      userId = telegramId
      localStorage.setItem("userId", userId)
      checkUserConnection()
      alert("Connected successfully! Your Telegram ID: " + userId)
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

  // Function to check if it's past 01:00 GMT+1
  function isPastResetTime(dateString) {
    if (!dateString) return true;
    
    const lastDate = new Date(dateString);
    const now = new Date();
    
    // Convert to GMT+1
    const nowGMT1 = new Date(now.getTime() + (1 * 60 * 60 * 1000));
    const resetTime = new Date(
      nowGMT1.getFullYear(),
      nowGMT1.getMonth(),
      nowGMT1.getDate(),
      1, 0, 0
    );
    
    // If current time is before 01:00, use yesterday's reset time
    if (nowGMT1.getHours() < 1) {
      resetTime.setDate(resetTime.getDate() - 1);
    }
    
    return lastDate < resetTime;
  }

  // Fetch user data from the API
  function fetchUserData() {
    if (!userId) return

    console.log("Fetching data for user ID:", userId)

    fetch(`https://binom-dots.onrender.com/api/user?id=${userId}`)
      .then((response) => {
        if (!response.ok) {
          throw new Error("User not found")
        }
        return response.json()
      })
      .then((data) => {
        console.log("User data received:", data)
        userDots = data.dots
        dotsCount.textContent = userDots

        // Check if daily rewards are available
        const checkInAvailable = isPastResetTime(data.last_check_in);
        const shareAvailable = isPastResetTime(data.last_share_reward);

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
        // If user not found, clear localStorage
        if (error.message === "User not found") {
          alert("User not found. Please connect with your Telegram ID again.")
          localStorage.removeItem("userId")
          checkUserConnection()
        }
      })
  }

  // Connect with Telegram
  telegramBtn.addEventListener("click", () => {
    // Open Telegram bot in a new window
    window.open("https://t.me/BinomDotsBot", "_blank")

    // Show instructions to the user
    alert(
      "1. Send /start to the bot\n2. Use /checkin and /share to earn dots\n3. Come back here and click 'Connect with Telegram ID'\n4. Enter your Telegram ID"
    )
  })

  // Claim daily check-in reward
  checkinBtn.addEventListener("click", () => {
    if (!userId) return

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
      const shareText = "💎 EXCLUSIVE: Collect Binom Dots daily and be first to claim $BINOM tokens! Limited opportunity: https://t.me/BinomChain_bot"

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