package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

type (
	Article struct {
		Title     string
		URL       string
		CreatedAt time.Time
	}
	ArticleSlice []Article
)

const blogDomain = "ameblo.jp"

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func (r ArticleSlice) Len() int {
	return len(r)
}

func (r ArticleSlice) Less(i, j int) bool {
	return i < j
}

func (r ArticleSlice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("required blog name")
		os.Exit(1)
	}
	var minusDaysIndex int
	var minusDays int
	for i, v := range os.Args {
		if v == "-d" || v == "--minus_days" {
			minusDaysIndex = i
		} else if minusDaysIndex == i-1 {
			if num, err := strconv.Atoi(v); err == nil {
				minusDays = num
			}
		}
	}
	run(os.Args[1], minusDays)
}

func run(name string, minusDays int) {
	y, m, d := time.Now().Date()
	d -= minusDays
	start := time.Date(y, m, d, 0, 0, 0, 0, jst)
	end := time.Date(y, m, d+1, 0, 0, 0, -1, jst)
	list, err := fetchArticleList(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	articles := make([]Article, 0, len(list))
	for _, article := range list {
		if article.CreatedAt.Before(start) || article.CreatedAt.After(end) {
			continue
		}
		articles = append(articles, article)
	}
	if len(articles) == 0 {
		fmt.Println("Not found the download target.")
		return
	}
	fmt.Println("Which article do you want to download? Please choose a number.")
	fmt.Println("You can specify more than one by separating with commas.")
	selectionMap := make(map[string]bool, len(articles))
	for i, article := range articles {
		index := fmt.Sprint(i + 1)
		selectionMap[index] = true
		fmt.Printf("- %d: %s %s\n", i+1, article.CreatedAt.Format("2006-01-02 15:04:05"), article.Title)
	}
	fmt.Print("selection number: ")
	var selection string
	if _, err := fmt.Scan(&selection); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if selection != "all" {
		numbers := strings.Split(selection, ",")
		selectionMap = make(map[string]bool, len(numbers))
		for _, n := range numbers {
			selectionMap[n] = true
		}
	}
	for i, article := range articles {
		if _, ok := selectionMap[fmt.Sprint(i+1)]; !ok {
			continue
		}
		fmt.Printf("download to %d.%s > ", i, article.Title)
		succeedFunc := func() {
			fmt.Printf("*")
		}
		if _, err := fetchArticle(article, succeedFunc); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("")
	}
	if err := exec.Command("open", fmt.Sprintf("./download/%04d%02d%02d", y, m, d)).Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func fetchArticle(v Article, succeedFunc func()) (int, error) {
	succeed := 0
	if len(v.URL) < 10 {
		return 0, nil
	}
	if v.URL[0:4] != "http" {
		v.URL = fmt.Sprintf("https://%s/%s", blogDomain, v.URL)
	}
	doc, _ := goquery.NewDocument(v.URL)
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		url, _ := s.Attr("src")
		// "https://stat.ameba.jp/user_images/"
		// "20171122/19/yotsuba-kids/78/81/j/t02200165_0480036014075936377.jpg?cpd=110"
		if strings.Index(url, "https://stat.ameba.jp/user_images/") != 0 {
			// fmt.Println(fmt.Sprintf("[NG] %s", v.URL))
			return
		}
		ok, err := downloadImage(url, "download/"+v.CreatedAt.Format("20060102"))
		if err != nil {
			fmt.Println(fmt.Sprintf("[ERROR] %+v, url=%s", err, url))
		}
		if ok {
			succeed++
			if succeedFunc != nil {
				succeedFunc()
			}
		}
	})
	return succeed, nil
}

// 記事一覧
func fetchArticleList(name string) ([]Article, error) {
	url := fmt.Sprintf("https://%s/%s/entrylist.html", blogDomain, name)
	doc, _ := goquery.NewDocument(url)
	articles := make([]Article, 0, 20)
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
		format := "2006-01-02 15:04:05"                      //Z07:00"
		t, err := time.ParseInLocation(format, timeStr, jst) //+"Z09:00")
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
	// sort to reverse
	for i := range articles {
		if len(articles)/2 < i {
			break
		}
		j := len(articles) - i - 1
		articles[i], articles[j] = articles[j], articles[i]
	}
	return articles, nil
}

func downloadImage(imageURL, savepath string) (bool, error) {

	resp, err := http.Get(imageURL)
	if err != nil {
		return false, errors.Wrapf(err, "failed to http get image, url=%s", imageURL)
	}
	if resp.StatusCode != 200 {
		return false, errors.Wrapf(err, "failed to http status, code=%d", resp.StatusCode)
	}
	if resp.ContentLength < 10000 {
		fmt.Println("[INFO] ignore contentLength min:", resp.ContentLength, imageURL)
		return false, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Wrapf(err, "failed to http get image stream, url=%s", imageURL)
	}

	_, filename := path.Split(imageURL)
	wk := strings.Split(filename, "?")
	filename = wk[0] // remove to query string

	// make dir
	if err := os.MkdirAll("./"+savepath, 0755); err != nil {
		return false, errors.Wrapf(err, "failed to create dir, path=%s", savepath)
	}

	file, err := os.OpenFile("./"+savepath+"/"+filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return false, errors.Wrapf(err, "failed to open file error, path=%s", "./"+savepath+"/"+filename)
	}

	defer func() {
		// conf, _, err := image.DecodeConfig(file)
		// if err != nil {
		// 	log.Fatal("image-error", err)
		// }
		// fmt.Printf("file=%s,Width=%d,Height=%d\n", filename, conf.Width, conf.Height)
		file.Close()
	}()

	if _, err := file.Write(body); err != nil {
		return false, errors.Wrapf(err, "failed to write file")
	}
	return true, nil
}
