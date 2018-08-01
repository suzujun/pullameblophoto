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
func (rs ArticleSlice) Len() int {
	return len(rs)
}

// Less ...
func (rs ArticleSlice) Less(i, j int) bool {
	return rs[i].CreatedAt.Unix() < rs[j].CreatedAt.Unix()
}

// Swap ...
func (rs ArticleSlice) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}
