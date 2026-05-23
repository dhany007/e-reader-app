# Aksara

Self-hosted AI-powered e-reader that translates English PDF books into natural Indonesian and serves them as a clean, mobile-friendly HTML reading experience.

## Features

- Upload English PDF books
- Automatic translation to Indonesian via DeepSeek API
- Preserves code blocks and software engineering terms
- Clean HTML reader (not a PDF viewer)
- Resumes reading from last position
- Dark mode
- Single-user, self-hosted

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go + Echo |
| AI | DeepSeek API (`deepseek-chat`) |
| PDF Extraction | Python + PyMuPDF |
| Frontend | Server-rendered HTML + Tailwind CSS |
| Database | SQLite |
| Deployment | Docker Compose |

## Architecture Overview

```
Upload PDF
    └─→ store to storage/pdfs/
    └─→ background pipeline:
        1. Python subprocess (PyMuPDF) → extract text per page → JSON
        2. Go worker → translate each page via DeepSeek API
        3. Save translated HTML fragments to SQLite
    └─→ book available in library

Open Book
    └─→ render HTML reader
    └─→ lazy load pages via fetch
    └─→ restore last scroll position
    └─→ save scroll position (debounced, every 2s)
```

## Project Structure

```
ai-reader/
├── cmd/server/main.go          # entry point
├── internal/
│   ├── config/                 # env config loader
│   ├── db/                     # SQLite connection + migrations
│   ├── handler/                # Echo HTTP handlers
│   ├── middleware/             # session auth middleware
│   ├── model/                  # data structs
│   ├── service/                # business logic + DeepSeek client
│   └── worker/                 # background translation pipeline
├── parser/extract.py           # PDF extractor (called as subprocess)
├── web/
│   ├── templates/              # HTML templates
│   └── static/                 # CSS + JS
├── storage/                    # runtime data (gitignored)
│   ├── pdfs/
│   └── html/
├── .env.example
├── Dockerfile
└── docker-compose.yml
```

## Database Schema

```sql
books             -- id, title, filename, category, status, total_pages, done_pages
pages             -- id, book_id, page_number, html_content
reading_progress  -- id, book_id, scroll_pct
```

Book `status` lifecycle: `pending → extracting → translating → done | error`

## API Routes

```
POST   /login
POST   /logout
GET    /library
POST   /books/upload
GET    /books/:id/status       JSON: progress polling
DELETE /books/:id
GET    /books/:id/read
GET    /books/:id/pages/:num   JSON: lazy load page HTML
POST   /books/:id/progress     JSON: save scroll position
GET    /books/:id/progress     JSON: restore scroll position
```

## Setup

### Prerequisites

- Docker + Docker Compose
- DeepSeek API key (https://api-docs.deepseek.com)

### 1. Clone and configure

```bash
git clone <repo>
cd ai-reader
cp .env.example .env
```

Edit `.env`:

```env
DEEPSEEK_API_KEY=sk-xxx
DEEPSEEK_MODEL=deepseek-chat
SESSION_SECRET=change-this-to-a-random-string
ADMIN_USERNAME=admin
ADMIN_PASSWORD_HASH=$2a$10$...   # bcrypt hash of your password
PORT=8080
DATA_DIR=./data
```

To generate a bcrypt hash for your password:

```bash
# using htpasswd
htpasswd -bnBC 10 "" yourpassword | tr -d ':\n'
```

### 2. Run

```bash
docker compose up -d
```

Open [http://localhost:8080](http://localhost:8080)

### Development (without Docker)

Requirements: Go 1.22+, Python 3.10+, pip

```bash
pip install pymupdf
go run ./cmd/server
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DEEPSEEK_API_KEY` | yes | — | DeepSeek API key |
| `DEEPSEEK_MODEL` | no | `deepseek-chat` | Model to use |
| `SESSION_SECRET` | yes | — | Random string for cookie signing |
| `ADMIN_USERNAME` | yes | — | Login username |
| `ADMIN_PASSWORD_HASH` | yes | — | bcrypt hash of login password |
| `PORT` | no | `8080` | HTTP port |
| `DATA_DIR` | no | `./data` | Path for SQLite DB |

## Out of Scope (MVP)

- Scanned PDF / OCR
- Multi-user
- Search inside book
- Export / download translated content
- Re-translation (translations are cached permanently)
- Epub or other formats

## Roadmap

| Sprint | Focus |
|--------|-------|
| 1 | Scaffold + DB migration |
| 2 | Auth (login, session middleware) |
| 3 | PDF upload + Python extractor |
| 4 | Translation pipeline + progress bar |
| 5 | HTML reader + scroll position |
| 6 | Dark mode + categories + mobile polish |
| 7 | Dockerfile + Docker Compose |
