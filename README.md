<div align="center">
  <img src="img/Logo-3.png" alt="HL Finance" width="140" />

  <h1>HL Finance — Personal Finance Manager</h1>

  <p>A full-stack personal finance app: track income & expenses, set budgets, and visualize your money — with real authentication and a bilingual (EN/VI) UI.</p>

  <p>
    <img alt="Go" src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" />
    <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black" />
    <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white" />
    <img alt="MySQL" src="https://img.shields.io/badge/MySQL-8-4479A1?logo=mysql&logoColor=white" />
    <img alt="Docker" src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white" />
  </p>
</div>

---

## ✨ Live demo

**➡️ [Try it live](https://finance.hlcompany.id.vn)** — then click **“Try the demo”** on the login screen to jump straight in with a pre-populated sandbox account (no sign-up needed).

## Features

- 🔐 **Real authentication** — register / login with JWT (HS256) + bcrypt-hashed passwords; every record scoped to the signed-in user.
- 💸 **Transactions** — table view with per-column search, grouping (by category / by type), show-hide columns, and a live total (Income − Expense). Amounts are always positive; the type column drives the math.
- 📊 **Dashboard** — monthly balance, savings rate, a 6-month cash-flow area chart, and an expense-breakdown donut.
- 🎯 **Budgets** — per-category monthly limits with progress bars and over/near/within-budget states.
- 🏷️ **Categories** — create custom categories (color + monochrome icon); 10 sensible defaults seeded on sign-up.
- 🌐 **Bilingual UI (EN/VI)** — one-click language switch; default category names localize too.
- 🔔 **Toasts** — success/error feedback on every create/update/delete.
- 🧪 **One-click demo account** — recruiters can explore instantly with realistic sample data.
- 🛡️ **Hardened** — rate-limited auth endpoints, configurable CORS, secret-strength warning.

## Tech stack

| Layer | Tech |
|------|------|
| **Backend** | Go 1.26, Gin, `database/sql` + MySQL, JWT (`golang-jwt/jwt/v5`), bcrypt |
| **Frontend** | React 19, TypeScript, Vite 6, Tailwind CSS v4, React Router 7, Recharts, lucide-react |
| **Infra** | Multi-stage Docker (~35 MB image), Docker Compose (app + MySQL) |

## Architecture

A single production binary (`cmd/web`) serves **both** the JSON API under `/api/*` and the built React SPA (with history-mode fallback), so the whole app deploys as one artifact.

```
Browser ──▶ Go (Gin) server ──▶ MySQL
             ├─ /api/*   → JSON API (auth + CRUD)   internal/api
             └─ /*       → React SPA (frontend/dist)
```

- **`internal/api`** — self-contained API package: `db.go` (connect + auto-migrate), `auth.go` (JWT + bcrypt + one-click demo), `categories/transactions/budgets.go` (CRUD), `ratelimit.go`, `router.go`.
- **`frontend/src`** — `context/` (Auth, Data, I18n, Toast), `pages/` (Login, Register, Dashboard, Transactions, Budgets), `lib/` (`api.ts`, `analytics.ts`, `i18n.ts`, `format.ts`).
- All API responses use the envelope `{ code, message, data }`.

## Getting started (local)

**Prerequisites:** Go 1.26+, Node 20+, Docker (for MySQL).

```bash
# 1. Clone
git clone https://github.com/nqghuy1202/financal_management.git
cd financal_management

# 2. Configure env
cp .env.example .env      # edit JWT_SECRET / DB credentials

# 3. Start MySQL
docker compose up -d mysql_bp

# 4a. Run everything as one server (serves API + built frontend at :8080)
make serve                # builds frontend + Go binary, then runs it
# open http://localhost:8080

# 4b. …or run in dev mode (hot reload)
go run ./cmd/web          # backend  :8080
cd frontend && npm run dev # frontend :5173 (proxies /api → :8080)
```

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the server listens on |
| `STATIC_DIR` | `./frontend/dist` | Built frontend directory |
| `CORS_ORIGINS` | `http://localhost:5173` | Allowed origins (comma-separated) |
| `JWT_SECRET` | — | Secret for signing JWTs. **Required, no fallback** — the server refuses to start if this is empty or still a placeholder (`change-me`/`dev-secret`), even locally. Generate one with `openssl rand -hex 32`. |
| `BLUEPRINT_DB_HOST/PORT/DATABASE/USERNAME/PASSWORD` | — | MySQL connection |

The schema is created automatically on boot (`CREATE TABLE IF NOT EXISTS`).

## Deployment

The repo ships a production **Dockerfile** and **docker-compose.yml**.

```bash
# Whole stack (app + MySQL)
docker compose up --build -d      # → http://localhost:8080

# Or build just the image
docker build -t financal-management .
```

**VPS (production):** `docker-compose.prod.yml` + Nginx reverse proxy + Let's Encrypt — see [`DEPLOY.md`](DEPLOY.md) ("Cách 3").

**One-click hosting (Railway):** deploy the repo (auto-detects the Dockerfile), add a MySQL database, and map its connection vars to `BLUEPRINT_DB_*` plus a `JWT_SECRET`. See [`DEPLOY.md`](DEPLOY.md) for details.

## Project structure

```
cmd/web/            Production entrypoint (API + SPA)
internal/api/       Real MySQL-backed API (auth + CRUD)
frontend/           React + TypeScript SPA
Dockerfile          Multi-stage build (Node → Go → Alpine)
docker-compose.yml  app + MySQL
DEPLOY.md           Deployment guide
```

> `cmd/api`, `cmd/server`, and `internal/controller|service|repo` are earlier learning scaffolds and are **not** used by the production build.

---

<div align="center"><sub>Built with Go, React & TypeScript.</sub></div>
