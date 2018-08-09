package scraping

import (
	"time"
)

type (
	// Picture 写真
	Picture struct {
		Size      int64     `json:"size"`
		URL       string    `json:"url"`
		Width     int       `json:"width"`
		Height    int       `json:"height"`
		CreatedAt time.Time `json:"createdAt"`
	}
)
