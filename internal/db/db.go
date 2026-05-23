package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func Init(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "ai-reader.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	// Add shelf_id column to books if not present (idempotent ALTER)
	if _, err := db.Exec(`ALTER TABLE books ADD COLUMN shelf_id INTEGER REFERENCES shelves(id) ON DELETE SET NULL`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}
	return nil
}

const schema = `
CREATE TABLE IF NOT EXISTS shelves (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS books (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT    NOT NULL,
    filename    TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'pending',
    total_pages INTEGER NOT NULL DEFAULT 0,
    done_pages  INTEGER NOT NULL DEFAULT 0,
    error_msg   TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pages (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id       INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    page_number   INTEGER NOT NULL,
    raw_text      TEXT    NOT NULL DEFAULT '',
    html_content  TEXT    NOT NULL DEFAULT '',
    translated_at DATETIME,
    UNIQUE(book_id, page_number)
);

CREATE TABLE IF NOT EXISTS reading_progress (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id    INTEGER NOT NULL UNIQUE REFERENCES books(id) ON DELETE CASCADE,
    scroll_pct REAL    NOT NULL DEFAULT 0.0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pages_book_id ON pages(book_id);
`
