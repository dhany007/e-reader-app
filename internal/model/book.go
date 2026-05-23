package model

import "time"

type BookStatus string

const (
	StatusPending     BookStatus = "pending"
	StatusExtracting  BookStatus = "extracting"
	StatusTranslating BookStatus = "translating"
	StatusDone        BookStatus = "done"
	StatusError       BookStatus = "error"
)

type Book struct {
	ID         int64
	Title      string
	Filename   string
	Category   string
	Status     BookStatus
	TotalPages int
	DonePages  int
	ErrorMsg   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Page struct {
	ID           int64
	BookID       int64
	PageNumber   int
	RawText      string
	HTMLContent  string
	TranslatedAt *time.Time
}
