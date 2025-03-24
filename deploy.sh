#!/bin/bash

# Build the Go application
echo "Building Go application..."
go build -o dots-app

# Create systemd service file
echo "Creating systemd service file..."
cat > dots-app.service << EOF
[Unit]
Description=Dots Rewards App
After=network.target

[Service]
ExecStart=/path/to/dots-app
WorkingDirectory=/path/to/app
Restart=always
RestartSec=10
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=dots-app
User=your-username

[Install]
WantedBy=multi-user.target
EOF

echo "Deployment files prepared!"
echo ""
echo "To deploy on your server:"
echo "1. Upload all files to your server"
echo "2. Install Go if not already installed"
echo "3. Run: go mod init dots-app"
echo "4. Run: go mod tidy"
echo "5. Run: chmod +x deploy.sh"
echo "6. Run: ./deploy.sh"
echo "7. Copy dots-app.service to /etc/systemd/system/"
echo "8. Run: sudo systemctl enable dots-app"
echo "9. Run: sudo systemctl start dots-app"
echo "10. Set up Nginx or Apache to proxy requests to the app"

