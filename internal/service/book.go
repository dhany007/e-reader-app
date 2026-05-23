package service

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"aksara/internal/config"
	"aksara/internal/model"

	"github.com/google/uuid"
)

type BookService struct {
	db           *sql.DB
	storageDir   string
	pythonBin    string
	parserScript string
}

func NewBookService(db *sql.DB, cfg *config.Config) *BookService {
	return &BookService{
		db:           db,
		storageDir:   cfg.StorageDir,
		pythonBin:    cfg.PythonBin,
		parserScript: cfg.ParserScript,
	}
}

func (s *BookService) CoverPath(bookID int64) string {
	return filepath.Join(s.storageDir, "covers", fmt.Sprintf("%d.jpg", bookID))
}

func (s *BookService) extractCover(bookID int64, pdfPath string) {
	coverDir := filepath.Join(s.storageDir, "covers")
	if err := os.MkdirAll(coverDir, 0755); err != nil {
		return
	}
	cmd := exec.Command(s.pythonBin, s.parserScript, "--cover", pdfPath, s.CoverPath(bookID))
	if err := cmd.Run(); err != nil {
		log.Printf("cover extraction failed book %d: %v", bookID, err)
	}
}

func (s *BookService) Upload(file multipart.File, header *multipart.FileHeader, title string) (*model.Book, error) {
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
		ext := filepath.Ext(header.Filename)
		title = header.Filename[:len(header.Filename)-len(ext)]
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
		`INSERT INTO books (title, filename, status) VALUES (?, ?, ?)
		 RETURNING id, title, filename, status, total_pages, done_pages, created_at, updated_at`,
		title, filename, model.StatusPending,
	).Scan(&book.ID, &book.Title, &book.Filename, &book.Status,
		&book.TotalPages, &book.DonePages, &book.CreatedAt, &book.UpdatedAt)
	if err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("insert book: %w", err)
	}

	go s.extractCover(book.ID, destPath)

	return book, nil
}

func (s *BookService) List(shelfID int64) ([]*model.Book, error) {
	var rows *sql.Rows
	var err error

	if shelfID < 0 {
		// Uncategorized: shelf_id IS NULL
		rows, err = s.db.Query(
			`SELECT b.id, b.title, b.filename, b.shelf_id, COALESCE(sh.name,''), b.status,
			        b.total_pages, b.done_pages, COALESCE(b.error_msg,''), b.created_at, b.updated_at
			 FROM books b LEFT JOIN shelves sh ON sh.id = b.shelf_id
			 WHERE b.shelf_id IS NULL
			 ORDER BY b.created_at DESC`,
		)
	} else if shelfID > 0 {
		rows, err = s.db.Query(
			`SELECT b.id, b.title, b.filename, b.shelf_id, COALESCE(sh.name,''), b.status,
			        b.total_pages, b.done_pages, COALESCE(b.error_msg,''), b.created_at, b.updated_at
			 FROM books b LEFT JOIN shelves sh ON sh.id = b.shelf_id
			 WHERE b.shelf_id = ?
			 ORDER BY b.created_at DESC`, shelfID,
		)
	} else {
		// shelfID == 0: all books
		rows, err = s.db.Query(
			`SELECT b.id, b.title, b.filename, b.shelf_id, COALESCE(sh.name,''), b.status,
			        b.total_pages, b.done_pages, COALESCE(b.error_msg,''), b.created_at, b.updated_at
			 FROM books b LEFT JOIN shelves sh ON sh.id = b.shelf_id
			 ORDER BY b.created_at DESC`,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []*model.Book
	for rows.Next() {
		b := &model.Book{}
		if err := rows.Scan(&b.ID, &b.Title, &b.Filename, &b.ShelfID, &b.ShelfName, &b.Status,
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
		`SELECT b.id, b.title, b.filename, b.shelf_id, COALESCE(sh.name,''), b.status,
		        b.total_pages, b.done_pages, COALESCE(b.error_msg,''), b.created_at, b.updated_at
		 FROM books b LEFT JOIN shelves sh ON sh.id = b.shelf_id
		 WHERE b.id = ?`, id,
	).Scan(&b.ID, &b.Title, &b.Filename, &b.ShelfID, &b.ShelfName, &b.Status,
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
	os.Remove(filepath.Join(s.storageDir, "pdfs", book.Filename))
	os.Remove(s.CoverPath(id))
	_, err = s.db.Exec(`DELETE FROM books WHERE id = ?`, id)
	return err
}

func (s *BookService) MoveBook(bookID, shelfID int64) error {
	if shelfID == 0 {
		_, err := s.db.Exec(`UPDATE books SET shelf_id = NULL, updated_at = ? WHERE id = ?`, time.Now(), bookID)
		return err
	}
	_, err := s.db.Exec(`UPDATE books SET shelf_id = ?, updated_at = ? WHERE id = ?`, shelfID, time.Now(), bookID)
	return err
}

func (s *BookService) GetPage(bookID int64, pageNum int) (*model.Page, error) {
	p := &model.Page{}
	err := s.db.QueryRow(
		`SELECT id, book_id, page_number, html_content FROM pages
		 WHERE book_id = ? AND page_number = ?`, bookID, pageNum,
	).Scan(&p.ID, &p.BookID, &p.PageNumber, &p.HTMLContent)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *BookService) GetProgress(bookID int64) (float64, error) {
	var pct float64
	err := s.db.QueryRow(
		`SELECT scroll_pct FROM reading_progress WHERE book_id = ?`, bookID,
	).Scan(&pct)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return pct, err
}

func (s *BookService) SaveProgress(bookID int64, scrollPct float64) error {
	_, err := s.db.Exec(
		`INSERT INTO reading_progress (book_id, scroll_pct, updated_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(book_id) DO UPDATE SET scroll_pct = excluded.scroll_pct, updated_at = excluded.updated_at`,
		bookID, scrollPct,
	)
	return err
}

// --- Shelf methods ---

func (s *BookService) ListShelves() ([]*model.Shelf, error) {
	rows, err := s.db.Query(
		`SELECT sh.id, sh.name, COUNT(b.id), sh.created_at
		 FROM shelves sh
		 LEFT JOIN books b ON b.shelf_id = sh.id
		 GROUP BY sh.id
		 ORDER BY sh.name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shelves []*model.Shelf
	for rows.Next() {
		sh := &model.Shelf{}
		if err := rows.Scan(&sh.ID, &sh.Name, &sh.BookCount, &sh.CreatedAt); err != nil {
			return nil, err
		}
		shelves = append(shelves, sh)
	}
	return shelves, rows.Err()
}

func (s *BookService) CreateShelf(name string) (*model.Shelf, error) {
	sh := &model.Shelf{}
	err := s.db.QueryRow(
		`INSERT INTO shelves (name) VALUES (?) RETURNING id, name, created_at`, name,
	).Scan(&sh.ID, &sh.Name, &sh.CreatedAt)
	return sh, err
}

func (s *BookService) DeleteShelf(id int64) error {
	// Books on this shelf become uncategorized (shelf_id set to NULL via ON DELETE SET NULL)
	_, err := s.db.Exec(`DELETE FROM shelves WHERE id = ?`, id)
	return err
}

func (s *BookService) ResetForRetry(id int64) {
	s.db.Exec(`UPDATE books SET status = 'pending', error_msg = NULL, done_pages = 0, updated_at = ? WHERE id = ?`, time.Now(), id)
}

func (s *BookService) UpdateStatus(id int64, status model.BookStatus, errMsg string) {
	s.db.Exec(
		`UPDATE books SET status = ?, error_msg = ?, updated_at = ? WHERE id = ?`,
		status, errMsg, time.Now(), id,
	)
}
