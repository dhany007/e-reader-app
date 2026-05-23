package service

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"ai-reader/internal/model"

	"github.com/google/uuid"
)

type BookService struct {
	db         *sql.DB
	storageDir string
}

func NewBookService(db *sql.DB, storageDir string) *BookService {
	return &BookService{db: db, storageDir: storageDir}
}

func (s *BookService) Upload(file multipart.File, header *multipart.FileHeader, title, category string) (*model.Book, error) {
	magic := make([]byte, 4)
	if _, err := file.Read(magic); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if string(magic) != "%PDF" {
		return nil, fmt.Errorf("file is not a valid PDF")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}

	if title == "" {
		title = header.Filename
	}
	if category == "" {
		category = "Uncategorized"
	}

	pdfDir := filepath.Join(s.storageDir, "pdfs")
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return nil, fmt.Errorf("create pdf dir: %w", err)
	}

	filename := uuid.New().String() + ".pdf"
	destPath := filepath.Join(pdfDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("save file: %w", err)
	}

	book := &model.Book{}
	err = s.db.QueryRow(
		`INSERT INTO books (title, filename, category, status) VALUES (?, ?, ?, ?)
		 RETURNING id, title, filename, category, status, total_pages, done_pages, created_at, updated_at`,
		title, filename, category, model.StatusPending,
	).Scan(&book.ID, &book.Title, &book.Filename, &book.Category, &book.Status,
		&book.TotalPages, &book.DonePages, &book.CreatedAt, &book.UpdatedAt)
	if err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("insert book: %w", err)
	}

	return book, nil
}

func (s *BookService) List() ([]*model.Book, error) {
	rows, err := s.db.Query(
		`SELECT id, title, filename, category, status, total_pages, done_pages,
		        COALESCE(error_msg, ''), created_at, updated_at
		 FROM books ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []*model.Book
	for rows.Next() {
		b := &model.Book{}
		if err := rows.Scan(&b.ID, &b.Title, &b.Filename, &b.Category, &b.Status,
			&b.TotalPages, &b.DonePages, &b.ErrorMsg, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

func (s *BookService) GetByID(id int64) (*model.Book, error) {
	b := &model.Book{}
	err := s.db.QueryRow(
		`SELECT id, title, filename, category, status, total_pages, done_pages,
		        COALESCE(error_msg, ''), created_at, updated_at
		 FROM books WHERE id = ?`, id,
	).Scan(&b.ID, &b.Title, &b.Filename, &b.Category, &b.Status,
		&b.TotalPages, &b.DonePages, &b.ErrorMsg, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

func (s *BookService) Delete(id int64) error {
	book, err := s.GetByID(id)
	if err != nil || book == nil {
		return err
	}
	pdfPath := filepath.Join(s.storageDir, "pdfs", book.Filename)
	os.Remove(pdfPath)
	_, err = s.db.Exec(`DELETE FROM books WHERE id = ?`, id)
	return err
}

func (s *BookService) PDFPath(filename string) string {
	return filepath.Join(s.storageDir, "pdfs", filename)
}

// UpdateStatus is called by the pipeline to update book state.
func (s *BookService) UpdateStatus(id int64, status model.BookStatus, errMsg string) {
	s.db.Exec(
		`UPDATE books SET status = ?, error_msg = ?, updated_at = ? WHERE id = ?`,
		status, errMsg, time.Now(), id,
	)
}
