package main

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
	urlsChan := make(chan string)
	go func() {
		for _, url := range urls {
			urlsChan <- url
		}
		close(urlsChan)
	}()

	var wg sync.WaitGroup

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range urlsChan {
				// ... (unchanged code)

				content, err := fetchContent(url)
				if err != nil {
					log.Printf("Error fetching content for %s: %v", url, err)
					continue
				}
				summary, err := summarizeContent(content, summarySentences)
				if err != nil {
					log.Printf("Error summarizing content for %s: %v", url, err)
					continue
				}
				fmt.Printf("\nSummary for %s:\n%s\n", url, summary)
			}
		}()
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

	return urls, nil
}

func countWords(wordList []string, excludedWords map[string]bool) map[string]int {
	wordMap := make(map[string]int)
	for _, word := range wordList {
		word = strings.ToLower(word)
		if !excludedWords[word] {
			wordMap[word]++
		}
	}

	return wordMap
}

func createFrequencies(wordMap map[string]int) []WordFrequency {
	frequencies := []WordFrequency{}

	for word, count := range wordMap {
		frequencies = append(frequencies, WordFrequency{word, count})
	}

	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].count > frequencies[j].count
	})

	return frequencies
}

func printFrequencies(frequencies []WordFrequency, number int) {
	count := 0
	for i := 0; count < number && i < len(frequencies); i++ {
		word := strings.TrimSpace(frequencies[i].word)
		if word == "" {
			continue
		}

		fmt.Printf("%s:%d\n", word, frequencies[i].count)
		count++
	}
}

func extractWords(url string) ([]string, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	words := []string{}
	wordRegex := regexp.MustCompile(`\w+`)
	doc.Find("body").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		matches := wordRegex.FindAllString(text, -1)
		words = append(words, matches...)
	})

	return words, nil
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

func summarizeContent(content string, summarySentences int) (string, error) {
	if summarySentences < 1 {
		return "", errors.New("summarySentences should be greater than or equal to 1")
	}

	bag := tldr.New()
	summary, err := bag.Summarize(content, summarySentences)
	if err != nil {
		return "", err
	}
	return strings.Join(summary, " "), nil
}
