package scraping

import (
	"encoding/json"
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
	"github.com/robertkrimen/otto"
)

const blogDomain = "ameblo.jp"

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// 記事一覧取得
func FindArticleList(name string) ([]Article, error) {
	url := fmt.Sprintf("https://%s/%s/entrylist.html", blogDomain, name)
	doc, _ := goquery.NewDocument(url)
	var script string
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		if !strings.HasPrefix(s.Text(), "window.INIT_DATA") {
			return
		}
		end := strings.Index(s.Text(), ";")
		script = "var data=" + s.Text()[17:end]
	})
	vm := otto.New()
	_, err := vm.Run(script + `;
	var values = []
	var obj = data.entryState.entryMap;
	Object.keys(obj).forEach(function (key) {
		values.push({
			"Id": obj[key].entry_id,
			"Title": obj[key].entry_title,
			"CreatedAt": obj[key].entry_created_datetime,
		});
	});
	var jsonstr = JSON.stringify(values);
	/*values.forEach(function(v, i){
		console.log(">>>jsout", i, v.Id, v.Title, v.CreatedAt);
	});*/`)
	if err != nil {
		fmt.Println("failed to parse script error", err.Error())
		return nil, err
	}
	res, err := vm.Get("jsonstr")
	if err != nil {
		fmt.Println("failed to get script values error", err.Error())
		return nil, err
	}
	jsonstr, err := res.ToString()
	if err != nil {
		fmt.Println("failed to script values to string error", err.Error())
		return nil, err
	}
	values := []struct {
		ID        int64
		Title     string
		CreatedAt string
	}{}
	err = json.Unmarshal([]byte(jsonstr), &values)
	if err != nil {
		fmt.Println("failed to get script values error", err.Error())
		return nil, err
	}
	articles := make(ArticleSlice, len(values))
	for i, v := range values {
		t, err := time.Parse(time.RFC3339Nano, v.CreatedAt)
		if err != nil {
			fmt.Println("failed to parse createdAt error", err)
			os.Exit(1)
			return nil, err
		}
		articles[i] = Article{
			Title:     v.Title,
			URL:       fmt.Sprintf("https://%s/%s/entry-%d.html", blogDomain, name, v.ID),
			CreatedAt: t,
		}
	}
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
