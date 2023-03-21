package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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

// processURLs manages the concurrent processing of URLs
func processURLs(urls []string, excludedWords map[string]bool, threads, number, summarySentences int) {
	var sem = make(chan struct{}, threads)
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type result struct {
		url     string
		summary string
	}

	results := make(chan result)

	go func() {
		for res := range results {
			fmt.Printf("\nSummary for %s:\n%s\n", res.url, res.summary)
		}
	}()

	for _, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(url string) {
			defer func() {
				<-sem
				wg.Done()
			}()
			bag := tldr.New()
			content, err := fetchContent(ctx, url)
			if err != nil {
				log.Printf("Error fetching content for %s: %v", url, err)
				return
			}
			summary, err := summarizeContent(bag, content, summarySentences)
			if err != nil {
				log.Printf("Error summarizing content for %s: %v", url, err)
				return
			}
			results <- result{url: url, summary: summary}
		}(url)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
	wg.Wait()
	close(results) // Close the results channel after all goroutines are done
}

// loadExcludedWords reads the excluded words file and returns a map of excluded words
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

// loadURLs reads the URLs file and returns a slice of URLs
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

// formatURL validates and formats the URL with the correct protocol
func formatURL(url string) (string, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	resp, err := http.Head(url)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	return url, nil
}

// fetchContent fetches the content of the given URL and returns it as a string
func fetchContent(ctx context.Context, url string) (string, error) {
	formattedURL, err := formatURL(url)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, formattedURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
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

// summarizeContent generates a summary of the content using the tldr.Bag package
func summarizeContent(bag *tldr.Bag, content string, summarySentences int) (string, error) {
	if summarySentences < 1 {
		return "", errors.New("summarySentences should be greater than or equal to 1")
	}

	summary, err := bag.Summarize(content, summarySentences)
	if err != nil {
		return "", err
	}

	return strings.Join(summary, " "), nil
}
