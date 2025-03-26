#!/bin/bash

# Set up Git configuration
git config --global user.name "Render Bot"
git config --global user.email "render-bot@example.com"

# Configure Git to use the token for authentication
git config --global credential.helper store
echo "https://igo-used:${GITHUB_TOKEN}@github.com" > ~/.git-credentials

# Set up the remote repository URL
REPO_URL="https://github.com/igo-used/binom-dots.git"

# Check if we're in a detached HEAD state (common in Render deployments)
if git rev-parse --abbrev-ref HEAD | grep -q "HEAD"; then
  echo "Detected detached HEAD state, checking out main branch..."
  git fetch origin main
  git checkout main || git checkout -b main
fi

# Check if origin remote exists, if not add it
if ! git remote | grep -q "^origin$"; then
  echo "Setting up origin remote..."
  git remote add origin $REPO_URL
else
  echo "Updating origin remote URL..."
  git remote set-url origin $REPO_URL
fi

# Make sure the users.json file exists
if [ ! -f "users.json" ]; then
  echo "Initializing users.json..."
  echo "[]" > users.json
  git add users.json
  git commit -m "Initialize users.json"
  git push origin main
fi

echo "Git setup complete!"