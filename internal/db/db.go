package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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

	// SQLite is single-writer; one connection avoids SQLITE_BUSY errors
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS books (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT    NOT NULL,
    filename    TEXT    NOT NULL,
    category    TEXT    NOT NULL DEFAULT 'Uncategorized',
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
