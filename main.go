package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/suzujun/pullameblophoto/scraping"
)

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

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
	list, err := scraping.FindArticleList(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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
		succeedFunc := func() {
			fmt.Printf("*")
		}
		if _, err := scraping.FindArticle(article, succeedFunc); err != nil {
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
