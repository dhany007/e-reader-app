package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"aksara/internal/config"
	"aksara/internal/service"
)

type Pipeline struct {
	db           *sql.DB
	storageDir   string
	pythonBin    string
	parserScript string
	translator   *service.Translator
}

func NewPipeline(db *sql.DB, cfg *config.Config) *Pipeline {
	return &Pipeline{
		db:           db,
		storageDir:   cfg.StorageDir,
		pythonBin:    cfg.PythonBin,
		parserScript: cfg.ParserScript,
		translator:   service.NewTranslator(cfg.DeepSeekAPIKey, cfg.DeepSeekModel),
	}
}

type extractedPage struct {
	Page int    `json:"page"`
	Text string `json:"text"`
}

type extractResult struct {
	Title string          `json:"title"`
	Pages []extractedPage `json:"pages"`
}

// Process runs the full pipeline for a book. Meant to be called in a goroutine.
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

	// Step 1: extract text from PDF via Python subprocess
	p.setStatus(bookID, "extracting")
	result, err := p.extract(pdfPath)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	p.db.Exec(
		`UPDATE books SET total_pages = ?, done_pages = 0, updated_at = ? WHERE id = ?`,
		len(result.Pages), time.Now(), bookID,
	)

	for _, pg := range result.Pages {
		if err := p.saveRawPage(bookID, pg.Page, pg.Text); err != nil {
			return fmt.Errorf("save raw page %d: %w", pg.Page, err)
		}
	}

	// Step 2: translate each page via DeepSeek API
	p.setStatus(bookID, "translating")
	ctx := context.Background()

	for _, pg := range result.Pages {
		if pg.Text == "" {
			p.incrementDone(bookID)
			continue
		}

		html, err := p.translator.Translate(ctx, pg.Text)
		if err != nil {
			return fmt.Errorf("translate page %d: %w", pg.Page, err)
		}

		p.db.Exec(
			`UPDATE pages SET html_content = ?, translated_at = ? WHERE book_id = ? AND page_number = ?`,
			html, time.Now(), bookID, pg.Page,
		)
		p.incrementDone(bookID)
	}

	p.setStatus(bookID, "done")
	return nil
}

func (p *Pipeline) extract(pdfPath string) (*extractResult, error) {
	cmd := exec.Command(p.pythonBin, p.parserScript, pdfPath)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("python exit %d: %s", exitErr.ExitCode(), exitErr.Stderr)
		}
		return nil, err
	}

	var result extractResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse output: %w", err)
	}
	return &result, nil
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
	p.db.Exec(
		`UPDATE books SET status = 'error', error_msg = ?, updated_at = ? WHERE id = ?`,
		msg, time.Now(), bookID,
	)
}

func (p *Pipeline) saveRawPage(bookID int64, pageNum int, rawText string) error {
	_, err := p.db.Exec(
		`INSERT INTO pages (book_id, page_number, raw_text)
		 VALUES (?, ?, ?)
		 ON CONFLICT(book_id, page_number) DO UPDATE SET raw_text = excluded.raw_text`,
		bookID, pageNum, rawText,
	)
	return err
}

func (p *Pipeline) incrementDone(bookID int64) {
	p.db.Exec(
		`UPDATE books SET done_pages = done_pages + 1, updated_at = ? WHERE id = ?`,
		time.Now(), bookID,
	)
}
