import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/JesusIslam/tldr"
	"github.com/PuerkitoBio/goquery"
)

type WordFrequency struct {
	word  string
	count int
}

func main() {
	targets := flag.String("t", "", "targets file (newline per webpage to load)")
	exclude := flag.String("e", "", "exclude file (newline per word to exclude)")
	number := flag.Int("n", 10, "the number of most common words to output")
	threads := flag.Int("threads", 10, "the number of threads to use")
	summarySentences := flag.Int("s", 3, "the number of sentences in the summary")

	flag.Parse()

	if *targets == "" {
		log.Fatal("Error: Missing -t or -targets flag")
	}

	excludedWords, err := loadExcludedWords(*exclude)
	if err != nil {
		log.Fatalf("Error loading excluded words: %v", err)
	}

	urls, err := loadURLs(*targets)
	if err != nil {
		log.Fatalf("Error loading URLs: %v", err)
	}

	processURLs(urls, excludedWords, *threads, *number, *summarySentences)
}

func processURLs(urls []string, excludedWords map[string]bool, threads, number, summarySentences int) {
	var sem = make(chan struct{}, threads)
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(url string) {
			defer func() {
				<-sem
				wg.Done()
			}()
			bag := tldr.New()
			content, err := fetchContent(url)
			if err != nil {
				log.Printf("Error fetching content for %s: %v", url, err)
				return
			}
			summary, err := summarizeContent(bag, content, summarySentences)
			if err != nil {
				log.Printf("Error summarizing content for %s: %v", url, err)
				return
			}
			fmt.Printf("\nSummary for %s:\n%s\n", url, summary)
		}(url)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
	wg.Wait()
}

func loadExcludedWords(filename string) (map[string]bool, error) {
	excludedWords := make(map[string]bool)

	if filename == "" {
		return excludedWords, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		excludedWords[strings.ToLower(scanner.Text())] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return excludedWords, nil
}

func loadURLs(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	urls := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		urls = append(urls, url)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

func summarizeContent(bag *tldr.Bag, content string, summarySentences int) (string, error) {
	if summarySentences < 1 {
		return "", errors.New("summarySentences should be greater than or equal to 1")
	}

	summary, err := bag.Summarize(content, summarySentences)
	if err != nil {
		return "", err
	}

	var summaryBuilder strings.Builder
	for i, sentence := range summary {
		summaryBuilder.WriteString(sentence)
		if i < len(summary)-1 {
			summaryBuilder.WriteString(" ")
		}
	}

	return summaryBuilder.String(), nil
}

func fetchContent(url string) (string, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", err
	}

	var contentBuilder strings.Builder
	doc.Find("body").Find("*").Each(func(_ int, s *goquery.Selection) {
		if s.Children().Length() == 0 {
			contentBuilder.WriteString(s.Text())
			contentBuilder.WriteString(" ")
		}
	})

	return contentBuilder.String(), nil
}
