package scraping

import (
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"path"
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

// 記事内容取得
func FindArticle(v Article, succeedFunc func()) (int, error) {
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

func downloadImage(imageURL, savepath string) (bool, error) {

	_, filename := path.Split(imageURL)
	wk := strings.Split(filename, "?")
	filename = wk[0] // remove to query string

	// make dir
	if err := os.MkdirAll("./"+savepath, 0755); err != nil {
		return false, errors.Wrapf(err, "failed to create dir, path=%s", savepath)
	}

	outputPath := "./" + savepath + "/" + filename

	newImage, err := os.Create(outputPath)
	if err != nil {
		return false, errors.Wrapf(err, "failed to http get image, url=%s", imageURL)
	}
	defer newImage.Close()

	resp, err := http.Get(imageURL)
	if err != nil {
		return false, errors.Wrapf(err, "failed2 to http get image, url=%s", imageURL)
	}
	defer resp.Body.Close()

	_, err = io.Copy(newImage, resp.Body)
	if err != nil {
		return false, errors.Wrapf(err, "failed3 to http get image, url=%s", imageURL)
	}
	// fmt.Println("File size: ", b)

	img, err := os.Open(outputPath)
	if err != nil {
		return false, errors.Wrapf(err, "failed to http get image, url=%s", imageURL)
	}

	_, err = jpeg.Decode(img)
	if err != nil {
		log.Fatal("image-error1b:", err)
	}
	// fmt.Println(">>>img:", fmt.Sprintf("%+v", jpg.Bounds()))

	// file, err := os.Open(outputPath)
	// if err != nil {
	// 	log.Fatal("image-error1", err)
	// }
	// conf, _, err := image.DecodeConfig(img)
	// if err != nil {
	// 	log.Fatal("image-error2", err)
	// }
	// fmt.Printf("file=%s,Width=%d,Height=%d\n", filename, conf.Width, conf.Height)

	return true, nil
	//////////////////////////////

	// resp, err = http.Get(imageURL)
	// if err != nil {
	// 	return false, errors.Wrapf(err, "failed to http get image, url=%s", imageURL)
	// }
	// if resp.StatusCode != 200 {
	// 	return false, errors.Wrapf(err, "failed to http status, code=%d", resp.StatusCode)
	// }
	// if resp.ContentLength < 10000 {
	// 	fmt.Println("[INFO] ignore contentLength min:", resp.ContentLength, imageURL)
	// 	return false, nil
	// }

	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	return false, errors.Wrapf(err, "failed to http get image stream, url=%s", imageURL)
	// }

	// _, filename := path.Split(imageURL)
	// wk := strings.Split(filename, "?")
	// filename = wk[0] // remove to query string

	// // make dir
	// if err := os.MkdirAll("./"+savepath, 0755); err != nil {
	// 	return false, errors.Wrapf(err, "failed to create dir, path=%s", savepath)
	// }

	// file, err := os.OpenFile("./"+savepath+"/"+filename, os.O_CREATE|os.O_WRONLY, 0666)
	// if err != nil {
	// 	return false, errors.Wrapf(err, "failed to open file error, path=%s", "./"+savepath+"/"+filename)
	// }

	// defer func() {
	// 	// conf, _, err := image.DecodeConfig(file)
	// 	// if err != nil {
	// 	// 	log.Fatal("image-error", err)
	// 	// }
	// 	// fmt.Printf("file=%s,Width=%d,Height=%d\n", filename, conf.Width, conf.Height)
	// 	file.Close()
	// }()

	// if _, err := file.Write(body); err != nil {
	// 	return false, errors.Wrapf(err, "failed to write file")
	// }
	return true, nil
}
