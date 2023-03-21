# Succinct
This Go script takes a list of URLs and outputs the most common words for each URL along with a summary paragraph describing the website. The script uses the `github.com/JesusIslam/tldr` library for summarization.

## Usage

1. Install the required libraries:

   ```
   go get github.com/JesusIslam/tldr
   go get github.com/PuerkitoBio/goquery
   ```

2. Build the script:

   ```
   go build main.go
   ```

3. Run the script with the required flags:

   ```
   ./main -t <targets-file> [-e <exclude-file>] [-n <number-of-common-words>] [-threads <number-of-threads>] [-s <number-of-summary-sentences>]
   ```

   - `-t` or `--targets`: Targets file (newline per webpage to load)
   - `-e` or `--exclude`: Exclude file (newline per word to exclude) - optional
   - `-n`: The number of most common words to output - optional, default is 10
   - `--threads`: The number of threads to use - optional, default is 10
   - `-s`: The number of sentences in the summary - optional, default is 3

## Example

Create a `targets.txt` file with a list of URLs to process:

```
https://example.com
https://example.org
```

Create an `exclude.txt` file with a list of words to exclude:

```
the
and
```

Run the script:

```
./main -t targets.txt -e exclude.txt -n 10 -threads 10 -s 3
```
