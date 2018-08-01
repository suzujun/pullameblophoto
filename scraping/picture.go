package scraping

import (
	"time"
)

type (
	// Picture 写真
	Picture struct {
		Size      int64
		URL       string
		Width     int
		Height    int
		CreatedAt time.Time
	}
)
