<div align="center">

# SlipWise

*Real-time sports betting accumulator tracker, live match settlement engine, and bet slip management backend.*

[![Go Version](https://img.shields.io/badge/Go-%E2%89%A51.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Deployed on Render](https://img.shields.io/badge/Render-Deployed-46E3B7?logo=render&logoColor=white)](https://render.com)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL%2015%2B-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Echo](https://img.shields.io/badge/Router-Echo%20v5-00ADD8?logo=go&logoColor=white)](https://echo.labstack.com/)

[![API Documentation](https://img.shields.io/badge/OpenAPI-Swagger%202.0-85EA2D?logo=swagger&logoColor=black)](http://localhost:8080/swagger/index.html)
[![FCM](https://img.shields.io/badge/Push-Firebase%20FCM-FFCA28?logo=firebase&logoColor=black)](https://firebase.google.com/docs/cloud-messaging)
[![Resend](https://img.shields.io/badge/Email-Resend-000000?logo=resend&logoColor=white)](https://resend.com)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)](#license)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/niyiayooluwa/slipwise/pulls)

<p align="center">
  <a href="https://slipwise.niyiayo.com"><strong>Landing Page</strong></a> •
  <a href="#about">About</a> •
  <a href="#key-features">Features</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#tech-stack">Tech Stack</a> •
  <a href="#getting-started">Getting Started</a> •
  <a href="#usage">Usage</a> •
  <a href="#api-reference">API Reference</a> •
  <a href="#project-structure">Structure</a> •
  <a href="#roadmap">Roadmap</a> •
  <a href="#testing--quality-assurance">Testing</a> •
  <a href="#contributing">Contributing</a> •
  <a href="#license">License</a>
</p>

</div>

---

## Table of Contents

- [About](#about)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Environment Configuration](#environment-configuration)
  - [Database Migrations](#database-migrations)
- [Usage](#usage)
  - [Running the API Server](#running-the-api-server)
  - [Interactive Swagger UI](#interactive-swagger-ui)
  - [Market Translator CLI](#market-translator-cli)
- [API Reference](#api-reference)
  - [Authentication](#authentication)
  - [Tickets & Bet Tracking](#tickets--bet-tracking)
  - [User Analytics & Admin](#user-analytics--admin)
- [Project Structure](#project-structure)
- [Roadmap](#roadmap)
- [Testing & Quality Assurance](#testing--quality-assurance)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

---

## About

**SlipWise** is the core backend engine powering the SlipWise betting companion ecosystem. It does not operate as a sportsbook or take wagers. Instead, it eliminates the anxiety and friction of manually tracking multibet accumulator tickets by ingesting raw bookmaker codes, standardizing betting markets, tracking live match scores minute-by-minute, resolving ticket outcomes in real-time, and computing punter performance metrics.

Tracking dozens of matches across accumulator legs across fragmented score apps creates significant cognitive overhead. SlipWise addresses this with a high-throughput **market-level bucket evaluation strategy**: instead of evaluating thousands of individual user slips one-by-one, the background evaluator resolves matches and market types at the bucket level, runs single-pass batch updates on all matching selections, and fans out instant push alerts to affected ticket holders.

---

## Key Features

- ⚡ **Automated Slip Parsing**: Ingests raw booking codes (SportyBet, etc.) and translates complex market variations, odds, handicaps, and fixtures into unified schemas.
- 🎯 **Market-Centric Bucket Settlement**: Evaluates live outcomes at the market pick level rather than per-user, preventing database lock contention during high-volume match finishes.
- 🔔 **Real-Time Push Notification Engine**: Dispatches immediate alerts via Firebase Cloud Messaging (FCM HTTP v1) for ticket status changes (`WON`, `LOST`, `EARLY_WIN`, `VOID`).
- 📊 **Materialized Performance Stats**: Computes historical punter metrics including total slips tracked, overall stake, net profit, win rate, and return on investment (ROI).
- 🛡️ **Enterprise-Grade Authentication**: Email/password registration with 6-digit OTP verification, Google OAuth 2.0 verification, JWT access tokens (15-min TTL), and rotating refresh tokens with automatic token-family revocation on reuse.
- 🌐 **Proxy-Aware Rate Limiting**: Employs `httprate` middleware with CIDR-validated reverse proxy IP resolution (`TRUSTED_PROXY_CIDRS`) protecting sensitive auth endpoints behind Cloudflare, Nginx, or ALB.
- 🧹 **Orphan Scavenger Worker**: Autonomous background worker that cleans up stale unlinked preview booking codes and orphaned match entities to preserve storage efficiency.
- 🔍 **Diagnostic Translation CLI**: Includes a developer CLI (`cmd/translate`) to test, debug, and inspect bookmaker JSON response mappings offline without touching the database.

---

## Architecture

SlipWise is built as a clean, modular monolith in Go, cleanly separating HTTP transport, domain services, database persistence, and autonomous background workers.

```mermaid
flowchart TD
    subgraph Clients["Clients & Edge Network"]
        Flutter["📱 Flutter Client App"]
        Cloudflare["☁️ Cloudflare Edge Proxy"]
        Landing["🌐 Landing Page (slipwise.niyiayo.com)"]
    end

    subgraph API["Application & HTTP Transport Layer (Echo v5)"]
        Router["Echo v5 HTTP Engine"]
        MW["Middleware Pipeline\n(CORS, RequestID, Recover, Proxy-Aware RateLimiter)"]
        AuthH["Auth Handler & JWT Issuer"]
        TicketH["Betting & Ticket Handler"]
        AdminH["Admin Handler & RBAC"]
        Swagger["OpenAPI / Swagger UI"]
    end

    subgraph Workers["Autonomous Background Workers"]
        Poller["⏱️ Settlement Poller\n(60s Ticker)"]
        Evaluator["⚡ Live Match Evaluator\n(Bucket Resolution)"]
        Scavenger["🧹 Orphan Scavenger\n(Cleanup Job)"]
    end

    subgraph External["External Services"]
        FCM["🔔 Firebase Cloud Messaging (FCM)"]
        Resend["✉️ Resend Transactional Email"]
        SportyBet["🏟️ Sportsbook Endpoints"]
    end

    subgraph DB["Persistence Layer (PostgreSQL 15+)"]
        PG["🐘 Managed PostgreSQL (pgxpool)"]
        Schema["Normalized Schema + Goose Migrations\n(users, tickets, booking_selections, matches)"]
        SQLC["sqlc Type-Safe Generated Queries"]
    end

    Flutter -->|HTTPS / REST| MW
    Landing -->|Inspect API| Swagger
    MW --> Router
    Router --> AuthH
    Router --> TicketH
    Router --> AdminH

    TicketH -->|Parse Codes| Cloudflare
    Cloudflare -->|Scrape / Hydrate| SportyBet

    Poller -->|Fetch Live Scores| Cloudflare
    Poller --> Evaluator
    Evaluator -->|Batch Settle Selections| SQLC
    Evaluator -->|Trigger Push Alerts| FCM
    FCM -->|Push Notification| Flutter

    AuthH -->|Send Verification OTPs| Resend
    Scavenger -->|Prune Expired Codes| SQLC

    AuthH --> SQLC
    TicketH --> SQLC
    AdminH --> SQLC
    SQLC --> PG
    PG --- Schema
```

### Architectural Highlights

1. **Strict Layered Domain Isolation**: Handlers know only Services; Services know only Repositories and Domain models; Repositories wrap `sqlc`-generated queries.
2. **Bucket Evaluation Pattern**: Rather than iterating $N$ users $\times$ $M$ tickets when a goal is scored, SlipWise updates the matching market selection bucket in PostgreSQL once, then fans out notifications to all ticket holders.
3. **Resilient Token Rotation & Reuse Detection**: Refresh tokens are stored hashed. If an already-rotated token is presented, the system detects a potential replay attack and revokes the entire token family.
4. **Graceful OS Signal Draining**: Traps `SIGINT`/`SIGTERM` and initiates a 10-second request draining timeout to prevent connection drops during continuous deployment.

---

## Tech Stack

| Layer | Technology | Purpose |
|---|---|---|
| **Runtime & Language** | [Go 1.24+](https://go.dev) | High-performance, concurrent, type-safe execution |
| **HTTP Framework** | [Echo v5](https://echo.labstack.com) | Low-latency HTTP routing, middleware pipeline, and context handling |
| **Database** | [PostgreSQL 15+](https://www.postgresql.org) | Primary relational data store and ACID state machine |
| **Connection Pooling** | [pgx/v5](https://github.com/jackc/pgx) | Native PostgreSQL binary protocol toolkit with `pgxpool` |
| **Query Generation** | [sqlc](https://sqlc.dev) | Compiles raw SQL queries into type-safe, boilerplate-free Go code |
| **Migrations** | [Goose v3](https://github.com/pressly/goose) | Incremental, reversible database schema migration tooling |
| **Push Notifications** | [Firebase Cloud Messaging](https://firebase.google.com/docs/cloud-messaging) | Real-time push alert dispatch via official Google API SDK |
| **Email Delivery** | [Resend Go SDK](https://resend.com) | Transactional OTP delivery and verification emails |
| **Edge Proxy** | [Cloudflare Workers](https://workers.cloudflare.com) | Anti-bot bypass, header normalization, and edge rate-limit mitigation |
| **Rate Limiting** | [httprate](https://github.com/go-chi/httprate) | Sliding-window in-memory IP rate limiter with CIDR proxy validation |
| **API Documentation** | [Swagger / OpenAPI](https://swagger.io) | Automated Swagger 2.0 docs via `swaggo/swag` and `http-swagger` |
| **Hosting & Deployment** | [Render](https://render.com) | Production container runtime and managed web service |

---

## Getting Started

Follow these steps to set up and run the SlipWise backend locally.

### Prerequisites

Ensure you have the following installed on your machine:

- **Go**: `1.24` or higher (`go version`)
- **PostgreSQL**: `14` or higher (running locally or via Docker)
- **Make**: Standard GNU Make utility
- **Code Generation & Linting Binaries**: Installed via `make tools`:
  - `sqlc`
  - `swag`
  - `goose`
  - `staticcheck`
  - `goimports`

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/niyiayooluwa/slipwise.git
   cd slipwise
   ```

2. **Install code-generation and toolchain binaries:**
   ```bash
   make tools
   ```

3. **Download Go dependencies:**
   ```bash
   make tidy
   ```

4. **Prepare local environment variables:**
   ```bash
   cp .env.example .env
   ```

### Environment Configuration

Configure `.env` with your database credentials and service secrets:

| Variable | Description | Default / Example | Required |
| :--- | :--- | :--- | :---: |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/slipwise?sslmode=disable` | Yes |
| `JWT_SECRET` | Secret key for signing HMAC-SHA256 tokens | `dev-secret-change-me` | Yes |
| `PORT` | HTTP server port | `8080` | No |
| `RESEND_API_KEY` | Resend API key for transactional emails | `re_your-resend-api-key` | Yes |
| `RESEND_FROM_ADDRESS` | Verified sending domain email | `mail.slipwise.niyiayo.com` | Yes |
| `GOOGLE_CLIENT_ID` | OAuth Client ID for verifying Google logins | `your-google-client-id.apps.googleusercontent.com` | No |
| `CLOUDFLARE_WORKER_URL` | Endpoint for the edge scraper proxy | `https://your-worker.workers.dev` | Yes |
| `FEEDBACK_EMAIL` | Admin mailbox for `/auth/feedback` | `your_admin@domain.com` | No |
| `FIREBASE_CREDENTIALS_JSON` | Firebase service account JSON key content | `{"type": "service_account", ...}` | Yes |
| `TRUSTED_PROXY_CIDRS` | Comma-separated CIDR blocks for trusted proxies | _(empty for direct internet)_ | No |
| `CRON_SECRET` | Secret for manually triggering the settlement poller | _(empty)_ | No |

> [!IMPORTANT]
> A valid Firebase service account JSON key with Firebase Cloud Messaging Admin permissions must be provided via the `FIREBASE_CREDENTIALS_JSON` environment variable.

### Database Migrations

Run database migrations using Goose via Makefile targets:

```bash
# Export environment variables from .env
set -a; source .env; set +a

# Apply all pending migrations
make migrate-up

# Rollback the last applied migration (if needed)
make migrate-down
```

---

## Usage

### Running the API Server

Compile all code generators, run static checks, and launch the binary:

```bash
# Build to bin/server and run
make run
```

Alternatively, run directly with the Go toolchain:

```bash
go run ./cmd/server
```

Structured JSON logging starts immediately:

```json
{"time":"2026-09-11T08:00:00.000Z","level":"INFO","msg":"server starting","port":"8080"}
```

### Interactive Swagger UI

Once the server is running, explore and test the entire API specification directly in your browser:

- **Swagger UI**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- **Health Check**: [http://localhost:8080/health](http://localhost:8080/health)

### Market Translator CLI

Test how new or modified sportsbook JSON payloads are normalized into SlipWise match models without database interaction:

```bash
# Run translator against a sample SportyBet booking response
go run ./cmd/translate -in ai/sportybet_samples/codeShare.json
```

---

## API Reference

The server exposes standard RESTful endpoints under `/auth` and `/v1`. Most `/v1` endpoints require a Bearer JWT passed via the `Authorization: Bearer <token>` header.

### Authentication

| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :---: |
| `POST` | `/auth/signup` | Register new user with email & password | None |
| `POST` | `/auth/verify` | Verify email with 6-digit OTP code | None |
| `POST` | `/auth/resend-otp` | Request a new verification OTP | None |
| `POST` | `/auth/login` | Authenticate and obtain JWT + Refresh Token | None |
| `POST` | `/auth/oauth/google` | Verify Google ID token and log in | None |
| `POST` | `/auth/refresh` | Exchange valid refresh token for new access token | None |
| `POST` | `/auth/logout` | Revoke active refresh token | None |
| `GET` | `/auth/check-username` | Check handle availability | None |
| `GET` | `/auth/me` | Fetch authenticated user profile | Bearer |
| `PATCH` | `/auth/me` | Update display name or username handle | Bearer |
| `POST` | `/auth/devices` | Register FCM push device token | Bearer |
| `POST` | `/auth/feedback` | Submit user feedback to admin team | Bearer |

### Tickets & Bet Tracking

| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :---: |
| `POST` | `/v1/tickets/preview` | Ingest booking code and return normalized match breakdown without saving | Bearer |
| `POST` | `/v1/tickets/track` | Save booking code to user's tracked slips and start monitoring | Bearer |
| `GET` | `/v1/tickets` | List user's active tracked tickets (paginated, status filter, delta syncs) | Bearer |
| `GET` | `/v1/tickets/archived` | List user's archived tickets (paginated, status filter, delta syncs) | Bearer |
| `POST` | `/v1/tickets/archive` | Bulk archive tickets by UUIDs, hiding from active feed & muting notifications | Bearer |
| `POST` | `/v1/tickets/unarchive` | Bulk restore archived tickets back to active dashboard | Bearer |
| `POST` | `/v1/tickets/delete` | Bulk soft-delete tickets while preserving immutable accounting & stats | Bearer |
| `GET` | `/v1/tickets/:id` | Get detailed breakdown of a ticket and all selection legs | Bearer |
| `PATCH` | `/v1/tickets/:id` | Edit ticket metadata (e.g., custom label or notes) | Bearer |
| `DELETE` | `/v1/tickets/:id` | Soft-delete single ticket (hides from feed, mutes alerts, preserves stats) | Bearer |

#### Preview Ticket Request Example

```bash
curl -X POST http://localhost:8080/v1/tickets/preview \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "bookmaker": "SPORTYBET",
    "booking_code": "BC123XYZ"
  }'
```

### User Analytics & Admin

| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :---: |
| `GET` | `/v1/users/me/stats` | Retrieve user betting statistics (win rate, streaks, ROI) | Bearer |
| `GET` | `/v1/admin/dashboard` | High-level system-wide telemetry and platform figures | Admin |
| `GET` | `/v1/admin/users` | Paginated user management table | Admin |
| `POST` | `/internal/cron/settle` | Trigger manual settlement poll cycle | `X-Cron-Secret` |

---

## Project Structure

```text
slipwise/
├── ai/                      # Product specifications, SRS documents, and sample sportsbook payloads
├── cmd/
│   ├── server/              # Application entrypoint (main.go) and dependency wiring
│   └── translate/           # Diagnostic CLI for testing sportsbook payload parsers
├── docs/                    # OpenAPI 2.0 definitions and generated Swagger files
├── internal/
│   ├── admin/               # Admin dashboards, telemetry handlers, and RBAC middlewares
│   ├── apitypes/            # Shared JSON request/response DTOs
│   ├── auth/                # JWT issuance, Google OAuth, password hashing, and user handlers
│   ├── betting/             # Core betting logic, market parsers, models, and repositories
│   │   └── provider/        # Bookmaker integration modules (e.g. SportyBet)
│   ├── config/              # Environment variable parsing and validation
│   ├── db/
│   │   └── generated/       # sqlc-generated type-safe database queries (do not edit)
│   ├── httpserver/          # Echo router wiring, rate limiting, and middleware pipeline
│   ├── mailer/              # Email delivery adapters (Resend)
│   ├── notification/        # Push notification delivery adapters (Firebase FCM)
│   └── worker/              # Background settlement poller, evaluator, and orphan scavenger
├── migrations/              # Incremental SQL migrations executed by Goose
├── query/                   # Raw SQL queries compiled by sqlc
├── redundancy/              # Direct Go client bypassing Cloudflare (HuggingFace / WAF fallback)
├── Makefile                 # Development tasks, build steps, and migration targets
├── sqlc.yaml                # sqlc code generation configuration
└── staticcheck.conf         # Static code analysis rules
```

---

## Roadmap

SlipWise's rollout strategy is divided into strategic milestones from core loop validation to full social monetization:

### Phase 1: Foundation & Core Loop (📍 Current Status: MVP Complete)
*Focus: Prove the core betting loop, eliminate tracking anxiety, and deliver instant settlement alerts.*
- [x] Automated SportyBet booking code ingestion and bet preview
- [x] Market-bucket settlement poller and live evaluator
- [x] Leg-by-leg status verification (`WON`, `LOST`, `PENDING`, `EARLY_WIN`, `VOID`)
- [x] Real-time push notification delivery via Firebase Cloud Messaging (FCM HTTP v1)
- [x] Materialized user performance statistics (win rate, ROI, total staked, streaks)
- [x] Admin dashboard telemetry and user inspection APIs
- [x] Orphan scavenger background worker for unlinked preview codes
- [ ] Admin manual settlement endpoint (force-settling stuck matches)
- [ ] Backend delta syncs (`?since=timestamp`) for instant mobile UI hydration

### Phase 2: Growth & Retention (The Guerrilla Phase)
*Focus: Hook users through competition, recognition, and zero-budget prestige mechanics.*
- [ ] **Gamification & Win Streaks**: Real-time XP, betting streaks, and achievement badges
- [ ] **Leaderboards**: Dynamic user rankings by XP, active win streaks, and net profit
- [ ] **Marketing Broadcast Engine**: Admin endpoint to broadcast custom push notifications to all users for trending high-odds slips
- [ ] **Punter Verification**: Admin verification badges to build prestige for top-performing bettors

### Phase 3: The Social Hub & Infrastructure Upgrade
*Focus: Transition from a single-player utility to a real-time community hub and reduce infrastructure costs.*
- [ ] **Centrifugo Integration**: Replace frontend HTTP polling with bi-directional WebSockets for instant live score pushes
- [ ] **Zero-Budget Data Pipeline**: Direct daily fixture hydration + raw WebSocket score interception (retiring fragile scraper fallbacks)
- [ ] **Public Timeline**: Global feed of recently tracked and trending booking codes
- [ ] **Usernames & Following**: Claimable `@handles` and the ability to follow top-performing punters
- [ ] **Live Match Chat**: Real-time match rooms for active accumulator legs

### Phase 4: Monetization (The Punter Economy)
*Focus: Platform sustainability and revenue generation for power punters.*
- [ ] **Punter Communities**: Subscription-gated groups where top punters charge monthly fees for early access to slips
- [ ] **Paystack & Recurring Billing**: Webhook-driven recurring subscriptions with idempotency handling
- [ ] **Rewarded Analytics**: Ad-supported tier unlocking premium betting analytics and deep head-to-head statistics

---

## Testing & Quality Assurance

Maintain strict code quality before submitting pull requests:

```bash
# Run unit tests
make test

# Run code formatter, go vet, and staticcheck
make lint

# Run the complete clean-room pipeline (used in CI)
make ci
```

> [!TIP]
> `make ci` regenerates all `sqlc` models and Swagger specs from scratch, runs `goimports`, executes `go vet`, verifies symbols with `staticcheck`, and executes the full test suite.

---

## Contributing

Contributions are welcome! To contribute to SlipWise:

1. **Fork the repository** on GitHub.
2. **Create a feature branch**:
   ```bash
   git checkout -b feat/bet9ja-provider
   ```
3. **Commit your changes**:
   ```bash
   git commit -m "feat(betting): add parser support for Bet9ja booking codes"
   ```
4. **Ensure clean verification**:
   ```bash
   make ci
   ```
5. **Push to your branch**:
   ```bash
   git push origin feat/bet9ja-provider
   ```
6. **Open a Pull Request** against the `main` branch with a clear description of your changes.

---

## License

Copyright © 2026 SlipWise. All rights reserved.

Unauthorized copying, modification, distribution, or commercial use of this software via any medium is strictly prohibited. For licensing inquiries, please contact the maintainers.

---

## Contact

- **Author / Maintainer**: [Ayooluwa Niyi](https://github.com/niyiayooluwa)
- **Email / Inquiries**: `mail.slipwise.niyiayo.com`
- **Landing Page**: [https://slipwise.niyiayo.com](https://slipwise.niyiayo.com)
- **Backend Repository**: [https://github.com/niyiayooluwa/slipwise](https://github.com/niyiayooluwa/slipwise)
- **Client App Repository**: [https://github.com/niyiayooluwa/slipwise-app](https://github.com/niyiayooluwa/slipwise-app)
- **Issue Tracker**: [https://github.com/niyiayooluwa/slipwise/issues](https://github.com/niyiayooluwa/slipwise/issues)
