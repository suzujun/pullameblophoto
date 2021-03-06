package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/suzujun/pullameblophoto/scraping"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	minusDays := fs.Int("d", 0, "Minus days")
	searchMaxDays := fs.Int("max", 1, "Maximum number of pages to search")
	name := os.Args[1]
	fs.Parse(os.Args[2:])
	if len(name) == 0 {
		fmt.Fprintln(os.Stderr, "required blog name")
		os.Exit(1)
	}
	run(name, *minusDays, *searchMaxDays)
}

func run(name string, minusDays, searchMaxDays int) {
	y, m, d := time.Now().Date()
	d -= minusDays
	start := time.Date(y, m, d, 0, 0, 0, 0, jst)
	end := time.Date(y, m, d+1, 0, 0, 0, -1, jst)
	list := make([]scraping.Article, 0, 10)
	for i := 1; i <= searchMaxDays; i++ {
		as, err := scraping.FindArticleList(name, i)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		list = append(list, as...)
	}
	articles := make([]scraping.Article, 0, len(list))
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
		urls, err := scraping.FindPictureURL(article)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		pictures := make([]scraping.Picture, 0, len(urls))
		outputPath := fmt.Sprintf("download/%s", article.CreatedAt.Format("20060102"))

		ctx := context.Background()
		s := semaphore.NewWeighted(5)
		group, _ := errgroup.WithContext(ctx)

		for i := range urls {
			url := urls[i]
			createdAt := article.CreatedAt.Add(time.Second * time.Duration(i))
			group.Go(func() error {
				s.Acquire(context.Background(), 1)
				defer s.Release(1)
				pic, err := scraping.DownloadFile(url, outputPath, &createdAt)
				if err != nil {
					return err
				}
				fmt.Printf("*")
				pictures = append(pictures, *pic)
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		articles[i].Pictures = pictures
		fmt.Println("")
	}

	jsonPath := fmt.Sprintf("download/%s/export.json", start.Format("20060102"))
	exportJSON(jsonPath, articles)

	exec.Command("open", fmt.Sprintf("./download/%04d%02d%02d", y, m, d)).Run()
}

func exportJSON(exportPath string, articles []scraping.Article) error {

	// if raw, err := ioutil.ReadFile("./sample.json"); err == nil {
	// 	var before FeatureCollection
	// 	if err := json.Unmarshal(raw, &before); err == nil {
	// 		// merge data
	// 	}
	// }

	data := map[string]interface{}{
		"articles":  articles,
		"createdAt": time.Now(),
	}
	bs, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to json parse error")
	}

	return ioutil.WriteFile(exportPath, bs, 0644)
}
