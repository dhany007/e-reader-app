I want to build a self-hosted AI Reader application for personal use.

Core idea:
- Upload English PDF books
- AI will:
  1. extract PDF contents
  2. translate into Indonesian
  3. polish/rewrite the translation to sound natural
  4. preserve software engineering technical terms
  5. preserve code blocks without translating them
- Final output should NOT be a translated PDF.
- Instead, generate a clean HTML-based reading experience similar to Google Play Books.
- The reader should be mobile-friendly and comfortable on phones/tablets.
- When the app is closed and reopened, it should resume from the last reading position.
- All translation results should be cached/saved permanently to avoid retranslating.
- Single-user application only.
- Simple username/password authentication.
- Self-hosted.
- Focus on MVP first.

Tech stack:
- Backend: Go + Echo
- AI: DeepSeek API (OpenAI-compatible, model: deepseek-chat)
- PDF parser: Python + PyMuPDF
- Frontend: Server-rendered HTML + Tailwind CSS
- Database: SQLite
- Deployment: Docker Compose
- Single repository (monorepo)

Environment variables (.env):
- DEEPSEEK_API_KEY=sk-xxx        # required
- DEEPSEEK_MODEL=deepseek-chat   # default: deepseek-chat
- SESSION_SECRET=xxx             # required, random string for cookie signing
- ADMIN_USERNAME=xxx             # required
- ADMIN_PASSWORD_HASH=xxx        # bcrypt hash of admin password
- PORT=8080                      # optional, default 8080
- DATA_DIR=./data                # storage path for PDFs and generated HTML

Architecture:
- Echo as API server
- Go calls Python as subprocess (os/exec) for PDF extraction
- Python script prints extracted text to stdout in JSON format
- No separate Python HTTP service (keep MVP simple)
- Go worker for translation pipeline
- DeepSeek API as AI provider (HTTP client, OpenAI-compatible)
- API key loaded from environment variable at startup
- Generate translated HTML chapters
- Store original PDFs
- Store generated HTML
- Store reading progress

Application flow:
1. Login
2. Upload PDF
3. Background processing:
   - extracting (Python subprocess)
   - translating (DeepSeek API)
   - polishing
   - generating HTML
4. Show processing progress bar
5. Book appears in library
6. When opening a book:
   - render AI-generated HTML
   - NOT a PDF viewer
7. Automatically resume last reading position

PDF chunking strategy:
- Extract full text per-page via Python/PyMuPDF
- Translate page-by-page (one API call per page)
- Each page chunk stored independently in DB to allow resume/retry on failure
- Pages within the same logical chapter are merged into a single HTML section

Authentication:
- Session-based (cookie) with bcrypt password hashing
- Single user — credentials loaded from environment variables
- No registration flow needed

Reading position:
- Stored per-book as scroll percentage (0.0–1.0) in SQLite
- Updated via JS fetch on scroll event (debounced, every 2 seconds)
- Restored on book open via JS window.scrollTo

Desired project structure:

ai-reader/
├── internal/
│   ├── handler/
│   ├── service/
│   ├── model/
│   └── worker/
├── parser/
│   └── extract.py
├── storage/
├── web/
│   ├── templates/
│   └── static/
├── scripts/
├── .env.example
├── docker-compose.yml
├── go.mod
└── main.go

MVP Features:
- Simple login
- Upload PDF
- AI translation
- HTML reader
- Resume reading
- Single-level categories
- Dark mode
- Processing progress UI

Out of scope for MVP:
- Scanned PDF / OCR
- Multi-user
- Search inside book
- Export/download translated content
- Re-translation (once translated, cached permanently)
- Epub / other formats

I want you to help me design:
1. Complete project structure
2. Folder architecture
3. Initial Echo setup
4. SQLite schema
5. DeepSeek API integration (OpenAI-compatible client in Go)
6. PDF extraction flow (Python subprocess called from Go)
7. Translation pipeline with per-page chunking
8. HTML generation strategy
9. Reader architecture
10. Docker Compose setup (app + Python, no Ollama)
11. API routes
12. Data models
13. MVP architecture best practices
14. Step-by-step implementation plan
15. Prioritized roadmap to avoid over-engineering

Requirements:
- Clean but simple architecture
- Realistic for a solo developer
- Maintainable
- Not over-engineered
- Production-minded but still simple

Main focus:
- reading experience
- translation quality
- maintainability
- simplicity
