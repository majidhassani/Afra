# Deploy AgentVerse to Ubuntu with GitHub Actions

This project deploys cleanly with Docker Compose. The `develop` branch is wired
to `.github/workflows/deploy-develop.yml`.

## 1. Prepare the Ubuntu server

Run these commands on the server as a sudo-capable user:

```bash
sudo apt update
sudo apt install -y ca-certificates curl git ufw

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo tee /etc/apt/keyrings/docker.asc >/dev/null
sudo chmod a+r /etc/apt/keyrings/docker.asc

. /etc/os-release
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $VERSION_CODENAME stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker "$USER"
```

Log out and back in after adding the user to the `docker` group.

Create an app directory:

```bash
sudo mkdir -p /opt/afra
sudo chown -R "$USER:$USER" /opt/afra
```

Open the API port if you expose it directly:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 8080/tcp
sudo ufw enable
```

## 2. Create a deploy SSH key

On your local machine:

```bash
ssh-keygen -t ed25519 -C "github-actions-afra" -f ~/.ssh/afra_deploy
```

Add the public key to the Ubuntu server:

```bash
ssh-copy-id -i ~/.ssh/afra_deploy.pub USER@SERVER_IP
```

The private key content goes into GitHub Secrets as `SSH_PRIVATE_KEY`.

## 3. Add GitHub secrets

In GitHub:

`Settings -> Secrets and variables -> Actions -> New repository secret`

Required secrets:

```text
SSH_HOST=your.server.ip
SSH_USER=ubuntu
SSH_PORT=22
APP_DIR=/opt/afra
SSH_PRIVATE_KEY=<contents of ~/.ssh/afra_deploy>
PRODUCTION_ENV=<full .env content for the server>
```

Recommended `PRODUCTION_ENV` starter:

```env
APP_ENV=production
HTTP_PORT=8080

JWT_SECRET=replace-with-a-long-random-secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

POSTGRES_USER=casemind
POSTGRES_PASSWORD=replace-with-a-strong-db-password
POSTGRES_DB=casemind
AUTO_MIGRATE=true

LLM_PROVIDER=glm
LLM_BASE_URL=https://your-openai-compatible-endpoint/v1
LLM_API_KEY=replace-with-your-key
LLM_MODEL=glm-5.2
LLM_TIMEOUT_SECONDS=120
LLM_MAX_TOKENS=4096
LLM_TEMPERATURE=0.7

RATE_LIMIT_AUTH_PER_MINUTE=20
RATE_LIMIT_AGENT_PER_MINUTE=30
```

For a free deterministic smoke deployment, use:

```env
LLM_PROVIDER=mock
LLM_BASE_URL=
LLM_API_KEY=
```

## 4. Push and deploy

Push to `develop`:

```bash
git push -u origin develop
```

GitHub Actions will:

1. Run `go test ./...`
2. SSH into the Ubuntu server
3. Clone or update `/opt/afra`
4. Write `.env` from `PRODUCTION_ENV`
5. Run `docker compose up --build -d`
6. Check `/health`

## 5. Useful server commands

```bash
cd /opt/afra
docker compose ps
docker compose logs -f api
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

Swagger will be available at:

```text
http://SERVER_IP:8080/swagger
```
