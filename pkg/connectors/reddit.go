package connectors

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jschell12/replicateme/pkg/corpus"
)

// RedditImportOptions controls Reddit data archive import.
type RedditImportOptions struct {
	File  string // path to ZIP or extracted directory
	Since *time.Time
}

// ImportReddit reads comments and posts from a Reddit data archive.
func ImportReddit(opts RedditImportOptions) ([]corpus.RawMessage, error) {
	dir := opts.File

	if strings.HasSuffix(opts.File, ".zip") {
		dir = "/tmp/replicateme-reddit-export"
		if err := extractZip(opts.File, dir); err != nil {
			return nil, fmt.Errorf("extract reddit zip: %w", err)
		}
	}

	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("Reddit export not found at %s", dir)
	}

	var messages []corpus.RawMessage

	// import comments
	// columns: id,permalink,date,ip,subreddit,gildings,link,parent,body,media
	commentsFile := filepath.Join(dir, "comments.csv")
	if _, err := os.Stat(commentsFile); err == nil {
		rows, err := readCSV(commentsFile)
		if err == nil {
			for _, row := range rows {
				if len(row) < 9 {
					continue
				}
				id := row[0]
				dateStr := row[2]
				subreddit := row[4]
				body := row[8]

				if id == "" || body == "" || dateStr == "" {
					continue
				}

				ts, err := time.Parse(time.RFC3339, dateStr)
				if err != nil {
					ts, err = time.Parse("2006-01-02 15:04:05", dateStr)
					if err != nil {
						continue
					}
				}
				if opts.Since != nil && ts.Before(*opts.Since) {
					continue
				}

				messages = append(messages, corpus.RawMessage{
					ID:         fmt.Sprintf("reddit-comment-%s", id),
					Text:       cleanRedditText(body),
					Platform:   corpus.Reddit,
					Timestamp:  ts.UTC(),
					IsFromUser: true,
					Metadata: map[string]any{
						"type":      "comment",
						"subreddit": subreddit,
					},
				})
			}
		}
	}

	// import posts
	// columns: id,permalink,date,ip,subreddit,gildings,title,url,body,media
	postsFile := filepath.Join(dir, "posts.csv")
	if _, err := os.Stat(postsFile); err == nil {
		rows, err := readCSV(postsFile)
		if err == nil {
			for _, row := range rows {
				if len(row) < 9 {
					continue
				}
				id := row[0]
				dateStr := row[2]
				subreddit := row[4]
				title := row[6]
				body := ""
				if len(row) > 8 {
					body = row[8]
				}

				if id == "" || dateStr == "" {
					continue
				}

				var parts []string
				if title != "" {
					parts = append(parts, title)
				}
				if body != "" {
					parts = append(parts, body)
				}
				text := strings.Join(parts, "\n\n")
				if text == "" {
					continue
				}

				ts, err := time.Parse(time.RFC3339, dateStr)
				if err != nil {
					ts, err = time.Parse("2006-01-02 15:04:05", dateStr)
					if err != nil {
						continue
					}
				}
				if opts.Since != nil && ts.Before(*opts.Since) {
					continue
				}

				messages = append(messages, corpus.RawMessage{
					ID:         fmt.Sprintf("reddit-post-%s", id),
					Text:       cleanRedditText(text),
					Platform:   corpus.Reddit,
					Timestamp:  ts.UTC(),
					IsFromUser: true,
					Metadata: map[string]any{
						"type":      "post",
						"subreddit": subreddit,
						"title":     title,
					},
				})
			}
		}
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}

var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

func cleanRedditText(text string) string {
	cleaned := text
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", `"`)
	cleaned = strings.ReplaceAll(cleaned, "&#39;", "'")
	cleaned = mdLinkRe.ReplaceAllString(cleaned, "$1")
	return strings.TrimSpace(cleaned)
}

// readCSV reads a CSV file, skipping the header row.
func readCSV(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		rows = append(rows, record)
	}
	return rows, nil
}
