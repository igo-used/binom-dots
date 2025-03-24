document.addEventListener("DOMContentLoaded", () => {
  const checkinBtn = document.getElementById("checkin-btn")
  const shareBtn = document.getElementById("share-btn")
  const telegramBtn = document.getElementById("telegram-btn")
  const dotsCount = document.getElementById("dots-count")
  const shareOptions = document.getElementById("share-options")
  const shareButtons = document.querySelectorAll(".share-btn")

  let userId = localStorage.getItem("userId")
  let userDots = 0

  // Check if user is connected
  function checkUserConnection() {
    if (userId) {
      // User is connected, fetch their data
      fetchUserData()
      telegramBtn.textContent = "Connected with Telegram"
      telegramBtn.disabled = true
      checkinBtn.disabled = false
      shareBtn.disabled = false
    } else {
      // User is not connected
      checkinBtn.disabled = true
      shareBtn.disabled = true
    }
  }

  // Fetch user data from the API
  function fetchUserData() {
    if (!userId) return

    fetch(`https://binom-dots.onrender.com/api/user?id=${userId}`)
      .then((response) => {
        if (!response.ok) {
          throw new Error("User not found")
        }
        return response.json()
      })
      .then((data) => {
        userDots = data.dots
        dotsCount.textContent = userDots

        // Check if daily rewards are available
        const now = new Date()
        const lastCheckIn = new Date(data.last_check_in)
        const lastShareReward = new Date(data.last_share_reward)

        const checkInAvailable = !data.last_check_in || (now - lastCheckIn) / (1000 * 60 * 60) >= 24

        const shareAvailable = !data.last_share_reward || (now - lastShareReward) / (1000 * 60 * 60) >= 24

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
          localStorage.removeItem("userId")
          checkUserConnection()
        }
      })
  }

  // Connect with Telegram
  telegramBtn.addEventListener("click", () => {
    // Open Telegram bot in a new window
    window.open("https://t.me/BinomDotsBot", "_blank")

    // For demo purposes, we'll simulate a successful connection
    // In a real app, you'd implement Telegram Login Widget
    // https://core.telegram.org/widgets/login

    // Simulate user login after 3 seconds
    setTimeout(() => {
      // Generate a random user ID for demo
      userId = Math.floor(Math.random() * 1000000).toString()
      localStorage.setItem("userId", userId)

      // Update UI
      checkUserConnection()

      // Show success message
      alert("Successfully connected with Telegram!")
    }, 3000)
  })

  // Add this after the telegramBtn event listener
  // For local testing only - simulates a login without actually opening Telegram
  document.addEventListener("keydown", (event) => {
    // Press Ctrl+L to simulate login
    if (event.ctrlKey && event.key === "l") {
      userId = Math.floor(Math.random() * 1000000).toString()
      localStorage.setItem("userId", userId)
      checkUserConnection()
      alert("Simulated login successful! User ID: " + userId)
    }
  })

  // Claim daily check-in reward
  checkinBtn.addEventListener("click", () => {
    if (!userId) return

    // In a real app, you'd make an API call to the backend
    // For demo, we'll simulate a successful claim
    userDots += 10
    dotsCount.textContent = userDots

    checkinBtn.disabled = true
    checkinBtn.textContent = "Claimed Today"

    // Show success message
    alert("You claimed 10 dots for daily check-in!")
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
      const shareText = "I'm collecting Binom Dots from Binomena Blockchain! Join me: https://dbotblock29.site"

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

      // Simulate successful share after 5 seconds
      setTimeout(() => {
        userDots += 20
        dotsCount.textContent = userDots

        shareBtn.disabled = true
        shareBtn.textContent = "Shared Today"

        // Hide share options
        shareOptions.classList.remove("active")

        alert("Thanks for sharing! You earned 20 dots!")
      }, 5000)
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

