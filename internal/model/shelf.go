package model

import "time"

type Shelf struct {
	ID        int64
	Name      string
	BookCount int
	CreatedAt time.Time
}
