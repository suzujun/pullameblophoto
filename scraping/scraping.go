package scraping

import (
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

const blogDomain = "ameblo.jp"

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// 記事一覧取得
func FindArticleList(name string) ([]Article, error) {
	url := fmt.Sprintf("https://%s/%s/entrylist.html", blogDomain, name)
	doc, _ := goquery.NewDocument(url)
	articles := make(ArticleSlice, 0, 20)
	doc.Find(".contentsList li").Each(func(_ int, li *goquery.Selection) {
		var url, title string
		li.Find(".contentTitle").Each(func(_ int, s *goquery.Selection) {
			url, _ = s.Attr("href")
			title = s.Text()
			// fmt.Println(fmt.Sprintf("[content01] %s", url))
		})
		var timeStr string
		li.Find("time").Each(func(_ int, s *goquery.Selection) {
			timeStr = s.Text() // i.g. "2017-11-21 13:31:04"
			// fmt.Println(fmt.Sprintf("[content02] %s", s.Text()))
		})
		format := "2006-01-02 15:04:05" //Z07:00"
		t, err := time.ParseInLocation(format, timeStr, jst)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}
		articles = append(articles, Article{
			Title:     title,
			URL:       url,
			CreatedAt: t,
		})
	})
	sort.Sort(articles)
	return articles, nil
}

// FindPictureURL find picture url slice from article
func FindPictureURL(v Article) ([]string, error) {
	if len(v.URL) < 10 {
		return nil, nil
	}
	if v.URL[0:4] != "http" {
		v.URL = fmt.Sprintf("https://%s/%s", blogDomain, v.URL)
	}
	doc, err := goquery.NewDocument(v.URL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to new document for %d", v.URL)
	}
	urls := make([]string, 0, 20)
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		url, ok := s.Attr("src")
		if !ok {
			log.Fatal("not found is img src", i)
			return
		}
		if strings.Index(url, "https://stat.ameba.jp/user_images/") != 0 {
			return
		}
		urls = append(urls, url)
	})
	return urls, nil
}

// DownloadFile download for file
func DownloadFile(fileURL, savepath string, forceCreatedAt *time.Time) (*Picture, error) {

	_, filename := path.Split(fileURL)
	wk := strings.Split(filename, "?")
	filename = wk[0] // remove to query string

	// make dir
	if err := os.MkdirAll("./"+savepath, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to create dir, path=%s", savepath)
	}

	var prefix string
	if forceCreatedAt != nil {
		prefix = forceCreatedAt.Format("20060102T150405_")
	}
	outputPath := "./" + savepath + "/" + prefix + filename

	newImage, err := os.Create(outputPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create image, path=%s", outputPath)
	}
	defer newImage.Close()

	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to http get image, url=%s", fileURL)
	}
	defer resp.Body.Close()

	var pic Picture
	pic.URL = fileURL
	pic.CreatedAt = time.Now()

	writtenSize, err := io.Copy(newImage, resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to bytes copy, url=%s", fileURL)
	}
	pic.Size = writtenSize

	img, err := os.Open(outputPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open image, path=%s", outputPath)
	}
	jpg, err := jpeg.Decode(img)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode jpeg, path=%s", outputPath)
	}
	if forceCreatedAt != nil {
		createdAt := *forceCreatedAt
		if err := os.Chtimes(outputPath, createdAt, createdAt); err != nil {
			return nil, errors.Wrapf(err, "failed to chtimes, path=%s", outputPath)
		}
		pic.CreatedAt = createdAt
	}

	if m := jpg.Bounds().Max; m.X > 0 {
		pic.Width = m.Y
		pic.Height = m.X
	}

	return &pic, nil
}
