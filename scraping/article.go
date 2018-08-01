package scraping

import (
	"time"
)

type (
	// Article 記事
	Article struct {
		Title     string
		URL       string
		CreatedAt time.Time
	}
	// ArticleSlice 記事リスト
	ArticleSlice []Article
)

// Len ...
func (r ArticleSlice) Len() int {
	return len(r)
}

// Less ...
func (r ArticleSlice) Less(i, j int) bool {
	return i < j
}

// Swap ...
func (r ArticleSlice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
