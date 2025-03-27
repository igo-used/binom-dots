// Handle share button clicks
const shareButtons = document.querySelectorAll(".share-button")
const shareOptions = document.querySelector(".share-options")
const shareBtn = document.querySelector("#shareBtn")

shareButtons.forEach((button) => {
  button.addEventListener("click", () => {
    const platform = button.getAttribute("data-platform")
    let shareUrl = ""

    // Define the exact share text we want to use everywhere
    const shareText =
      "💎 EXCLUSIVE: Collect Binom Dots daily and be first to claim $BINOM tokens! Limited opportunity: https://t.me/BinomChain_bot"

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
            // Fallback - create a textarea element to copy text
            const textarea = document.createElement("textarea")
            textarea.value = shareText
            document.body.appendChild(textarea)
            textarea.select()
            document.execCommand("copy")
            document.body.removeChild(textarea)
            alert("Text copied! Open Instagram and paste to share.")
            window.open("https://www.instagram.com/", "_blank")
          })
        break

      case "telegram":
        // For Telegram, use the direct share URL without additional URL parameter
        shareUrl = `https://t.me/share/url?text=${encodeURIComponent(shareText)}`
        window.open(shareUrl, "_blank")
        break

      case "twitter":
        // For Twitter, just use the text parameter
        shareUrl = `https://twitter.com/intent/tweet?text=${encodeURIComponent(shareText)}`
        window.open(shareUrl, "_blank")
        break

      case "whatsapp":
        // For WhatsApp, use a simpler approach with just the text
        // Add line breaks to improve readability on WhatsApp
        const whatsappText = shareText.replace("! ", "!\n\n").replace("Limited opportunity: ", "Limited opportunity:\n")
        shareUrl = `https://wa.me/?text=${encodeURIComponent(whatsappText)}`
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

