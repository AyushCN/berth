# Berth single-host quickstart

Berth runs its API and worker as host processes. PostgreSQL, Redis, and NATS can run in Docker Compose. The worker needs access to the host containerd socket and a Linux machine; the full sandbox path does not run on Windows.

## Local development

Prerequisites: Go 1.26.3, Node.js/npm, Docker Compose, `migrate`, and a Linux host with containerd. Configure containerd for the `runc` runtime (the verified default) and ensure the worker user can access its socket.

1. Start dependencies: `make up`.
2. Create `backend/.env` or export environment variables. At minimum set `JWT_SECRET`, `ENCRYPTION_KEY` (32 raw bytes or 64 hex characters), `DATABASE_URL`, `REDIS_URL`, `NATS_URL`, `CONTAINERD_SOCK`, and `FRONTEND_URL`. Set the GitHub OAuth client variables to enable GitHub login and push.
3. Apply schema: `make migrate-up`.
4. In one terminal run `cd backend && go run ./cmd/api`.
5. In another run `cd backend && MODE=worker go run ./cmd/worker`.
6. In a third run `cd frontend && npm install && npm run dev`, then open `http://localhost:3000`.

In development, the API exposes a dev-login route. Production requires a strong `JWT_SECRET`, `ENCRYPTION_KEY`, and GitHub OAuth credentials. Do not reuse the development credentials from `infra/docker-compose.yml` in a public environment.

## Single-host VPS services

Install Docker Compose, Go 1.26.3, `migrate`, Git, and containerd on a Linux VPS. Configure containerd and create a dedicated Berth host account with access to the containerd socket. Keep the database and Redis ports bound to loopback; expose only the API and frontend through your TLS reverse proxy.

Create a root `.env` with unique `POSTGRES_PASSWORD` and `REDIS_PASSWORD`, then run `make prod` to start the stateful dependencies. Export `DATABASE_URL`, `REDIS_URL` (including its password), `NATS_URL`, `JWT_SECRET`, `ENCRYPTION_KEY`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `FRONTEND_URL`, `CONTAINERD_SOCK`, `WORKSPACE_ROOT`, and `ENV=production` for the API and worker. Run `make migrate-up`, build with `make build`, and manage `backend/bin/berth-api` and `backend/bin/berth-worker` with systemd. Build and serve the Next.js frontend with `cd frontend && npm ci && npm run build && npm run start`.

Check `GET /health` before sending traffic. It returns HTTP 503 if PostgreSQL or Redis is unhealthy. Berth is intended for trusted users on one host: sandbox processes currently use host networking, so do not expose this as a multi-tenant service.
