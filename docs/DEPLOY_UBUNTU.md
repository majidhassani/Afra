# Deploy AgentVerse to Ubuntu with GitHub Actions Self-Hosted Runner

This project deploys cleanly with Docker Compose. The `develop` branch is wired
to `.github/workflows/deploy-develop.yml`.

The current workflow uses a GitHub Actions self-hosted runner on the Ubuntu
server. It does not SSH into the server. The deploy job runs directly on a
Linux x64 self-hosted runner.

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

Create the server environment directory:

```bash
sudo mkdir -p /opt/afra
sudo chown -R "$USER:$USER" /opt/afra
```

Open the API and web ports if you expose them directly:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 8080/tcp
sudo ufw allow 3000/tcp
sudo ufw enable
```

## 2. Put the production environment on the server

Create:

```bash
nano /opt/afra/.env
```

Recommended starter:

```env
APP_ENV=production
HTTP_PORT=8080
WEB_PORT=3000

# Leave this empty in Docker production so the web container proxies /api to
# the backend service. Set it only when the browser must call another API URL.
VITE_API_BASE_URL=
VITE_ENABLE_MOCKS=false
VITE_DEFAULT_LANGUAGE=fa
VITE_APP_VERSION=production

JWT_SECRET=replace-with-a-long-random-secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

POSTGRES_USER=casemind
POSTGRES_PASSWORD=replace-with-a-strong-db-password
POSTGRES_DB=casemind
AUTO_MIGRATE=true

LLM_PROVIDER=mock
LLM_BASE_URL=
LLM_API_KEY=
LLM_MODEL=glm-5.2
LLM_TIMEOUT_SECONDS=120
LLM_MAX_TOKENS=4096
LLM_TEMPERATURE=0.7

RATE_LIMIT_AUTH_PER_MINUTE=20
RATE_LIMIT_AGENT_PER_MINUTE=30
```

Lock it down:

```bash
chmod 600 /opt/afra/.env
```

For GLM later, switch:

```env
LLM_PROVIDER=glm
LLM_BASE_URL=https://your-openai-compatible-endpoint/v1
LLM_API_KEY=replace-with-your-key
```

## 3. Install the GitHub Actions self-hosted runner

In GitHub:

```text
Repository -> Settings -> Actions -> Runners -> New self-hosted runner
```

Choose Linux x64 and follow GitHub's commands on the server.

Important:

- The runner name can be `majid`.
- `runs-on` matches labels, not the runner name. The workflow uses the default
  labels `self-hosted`, `Linux`, and `X64`.
- Run it as a service so it survives reboot.
- Do not place the runner directory inside the git repository.

If you already created it inside the repository, it is ignored by git via:

```text
actions-runner/
```

After configuring the runner, install it as a service from inside the runner
directory:

```bash
sudo ./svc.sh install
sudo ./svc.sh start
sudo ./svc.sh status
```

The runner must appear as online in:

```text
Settings -> Actions -> Runners
```

## 4. Optional GitHub secret

No SSH secrets are required with the self-hosted runner.

Optional secret:

```text
APP_ENV_FILE=/opt/afra/.env
```

If omitted, the workflow uses `/opt/afra/.env`. If that file is missing but
`.env` exists in the runner checkout workspace, the workflow falls back to that
workspace file.

Optional secret:

```text
PRODUCTION_ENV=<full env file content>
```

If set, the workflow writes this secret into the env file on each deploy.
If omitted, it uses the existing server-side `/opt/afra/.env`.

## 5. Push and deploy

Push to `develop`:

```bash
git push origin develop
```

Or manually run:

```text
Actions -> Deploy develop -> Run workflow
```

GitHub Actions will:

1. Run `go test ./...`
2. Run deploy on the self-hosted Linux x64 runner
3. Use `/opt/afra/.env`
4. Run `docker compose --env-file /opt/afra/.env -p afra up --build -d`
5. Check API `/health`
6. Check web `/health`

## 6. Useful server commands

```bash
docker compose -p afra ps
docker compose -p afra logs -f api
docker compose -p afra logs -f web
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:3000/health
```

Swagger will be available at:

```text
http://SERVER_IP:8080/swagger
```

The web app will be available at:

```text
http://SERVER_IP:3000
```

From the web app, API calls go through:

```text
http://SERVER_IP:3000/api/v1/...
```
