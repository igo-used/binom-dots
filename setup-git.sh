#!/bin/bash

# Set up Git configuration
git config --global user.name "Render Bot"
git config --global user.email "render-bot@example.com"

# Create a personal access token with repo scope on GitHub
# Then set it as an environment variable in Render called GITHUB_TOKEN

# Configure Git to use the token for authentication
git config --global credential.helper store
echo "https://igo-used:${GITHUB_TOKEN}@github.com" > ~/.git-credentials

# Make sure the repository is properly cloned
if [ ! -d ".git" ]; then
  git clone https://github.com/igo-used/binom-dots .
fi

# Make sure the users.json file exists
if [ ! -f "users.json" ]; then
  echo "[]" > users.json
  git add users.json
  git commit -m "Initialize users.json"
  git push origin main
fi

echo "Git setup complete!"

