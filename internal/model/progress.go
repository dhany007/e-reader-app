package model

import "time"

type ReadingProgress struct {
	ID        int64
	BookID    int64
	ScrollPct float64
	UpdatedAt time.Time
}
