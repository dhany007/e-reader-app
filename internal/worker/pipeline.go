package worker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"ai-reader/internal/config"
)

type Pipeline struct {
	db           *sql.DB
	storageDir   string
	pythonBin    string
	parserScript string
}

func NewPipeline(db *sql.DB, cfg *config.Config) *Pipeline {
	return &Pipeline{
		db:           db,
		storageDir:   cfg.StorageDir,
		pythonBin:    cfg.PythonBin,
		parserScript: cfg.ParserScript,
	}
}

type extractedPage struct {
	Page int    `json:"page"`
	Text string `json:"text"`
}

// Process runs the full pipeline for a book in a background goroutine.
func (p *Pipeline) Process(bookID int64) {
	if err := p.process(bookID); err != nil {
		log.Printf("pipeline error book %d: %v", bookID, err)
		p.setError(bookID, err.Error())
	}
}

func (p *Pipeline) process(bookID int64) error {
	filename, err := p.getFilename(bookID)
	if err != nil {
		return fmt.Errorf("get filename: %w", err)
	}

	pdfPath := filepath.Join(p.storageDir, "pdfs", filename)

	p.setStatus(bookID, "extracting")
	pages, err := p.extract(pdfPath)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	p.db.Exec(`UPDATE books SET total_pages = ?, updated_at = ? WHERE id = ?`,
		len(pages), time.Now(), bookID)

	for _, pg := range pages {
		if err := p.savePage(int(bookID), pg.Page, pg.Text); err != nil {
			return fmt.Errorf("save page %d: %w", pg.Page, err)
		}
	}

	// Translation is added in Phase 4.
	// For now mark done so the book appears in the library.
	p.setStatus(bookID, "done")
	return nil
}

func (p *Pipeline) extract(pdfPath string) ([]extractedPage, error) {
	cmd := exec.Command(p.pythonBin, p.parserScript, pdfPath)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("python exit %d: %s", exitErr.ExitCode(), exitErr.Stderr)
		}
		return nil, err
	}

	var pages []extractedPage
	if err := json.Unmarshal(out, &pages); err != nil {
		return nil, fmt.Errorf("parse output: %w", err)
	}
	return pages, nil
}

func (p *Pipeline) getFilename(bookID int64) (string, error) {
	var filename string
	err := p.db.QueryRow(`SELECT filename FROM books WHERE id = ?`, bookID).Scan(&filename)
	return filename, err
}

func (p *Pipeline) setStatus(bookID int64, status string) {
	p.db.Exec(`UPDATE books SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now(), bookID)
}

func (p *Pipeline) setError(bookID int64, msg string) {
	p.db.Exec(`UPDATE books SET status = 'error', error_msg = ?, updated_at = ? WHERE id = ?`,
		msg, time.Now(), bookID)
}

func (p *Pipeline) savePage(bookID, pageNum int, rawText string) error {
	_, err := p.db.Exec(
		`INSERT INTO pages (book_id, page_number, raw_text)
		 VALUES (?, ?, ?)
		 ON CONFLICT(book_id, page_number) DO UPDATE SET raw_text = excluded.raw_text`,
		bookID, pageNum, rawText,
	)
	if err != nil {
		return err
	}
	p.db.Exec(`UPDATE books SET done_pages = done_pages + 1, updated_at = ? WHERE id = ?`,
		time.Now(), bookID)
	return nil
}
