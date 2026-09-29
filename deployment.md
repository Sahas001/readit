# Deploying ReadIT to EC2

This guide walks you through deploying ReadIT on an AWS EC2 instance running Ubuntu.
ReadIT runs on port **2222** so that your standard admin SSH port **22** remains untouched.

Friends connect using:
```bash
ssh -p 2222 readit.phasethru.dev
```

---

## 1. Prerequisites on EC2

Connect to your EC2 instance via SSH:
```bash
ssh -i your-key.pem ubuntu@your-ec2-ip
```

### Install Docker & Go
```bash
sudo apt-get update
sudo apt-get install -y docker.io docker-compose make git

# Add ubuntu user to docker group
sudo usermod -aG docker ubuntu
newgrp docker

# Install Go 1.22+
sudo snap install go --classic
```

---

## 2. AWS Security Group Rules

In AWS EC2 Console → Instances → Security Groups → **Edit inbound rules**:
- **Port 22** (SSH): Admin management (Restricted to your IP or anywhere)
- **Port 2222** (Custom TCP): `0.0.0.0/0` (ReadIT forum for friends)

---

## 3. Clone Repository & Setup Environment

```bash
git clone https://github.com/sahas/ReadIT.git ~/readit
cd ~/readit

# Create .env configuration
cat <<'EOF' > .env
SSH_ADDRESS=0.0.0.0:2222
SSH_HOST_KEY_PATH=.ssh/host_key
DATABASE_URL=postgres://readit:readit_secret@127.0.0.1:5432/readit?sslmode=disable
LOG_LEVEL=info
EOF

# Create host key directory
mkdir -p .ssh
chmod 700 .ssh
```

---

## 4. Launch Database & Apply Migrations

```bash
# Start PostgreSQL 16 container
make up

# Run migrations (applies 00001 through 00008)
make migrate-up

# Build the server binary
make build
```

---

## 5. Setup Systemd Service (24/7 Uptime)

Create `/etc/systemd/system/readit.service`:

```bash
sudo bash -c 'cat <<EOF > /etc/systemd/system/readit.service
[Unit]
Description=ReadIT SSH Terminal Forum
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/home/ubuntu/readit
ExecStart=/home/ubuntu/readit/bin/readit-server
Restart=always
RestartSec=3
Environment="TERM=xterm-256color" "COLORTERM=truecolor"
EnvironmentFile=/home/ubuntu/readit/.env
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF'

# Reload systemd and start service
sudo systemctl daemon-reload
sudo systemctl enable readit
sudo systemctl restart readit

# Check status
sudo systemctl status readit
```

> [!NOTE]
> The `Environment="TERM=xterm-256color" "COLORTERM=truecolor"` directive ensures the background systemd service advertises TrueColor support to connected terminal clients.

---

## 6. How to Update to the Latest Version

Whenever changes are pushed to GitHub, update your server with:

```bash
cd ~/readit
git pull origin main
make build
sudo systemctl restart readit
```

---

## 7. Connecting to ReadIT

From any computer:
```bash
ssh -p 2222 readit.phasethru.dev
```
