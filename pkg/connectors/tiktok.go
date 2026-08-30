package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jschell12/replicateme/pkg/corpus"
)

// TikTokImportOptions controls TikTok data download import.
type TikTokImportOptions struct {
	File     string // path to ZIP or extracted directory
	Username string // your TikTok username to identify your messages
	Since    *time.Time
}

type tikTokCommentList struct {
	CommentList []struct {
		Date    string `json:"Date"`
		Comment string `json:"Comment"`
	} `json:"Comment List"`
}

type tikTokDMConversation struct {
	Messages []struct {
		Date    string `json:"Date"`
		From    string `json:"From"`
		Content string `json:"Content"`
	} `json:"messages"`
}

// ImportTikTok reads comments and DMs from a TikTok data download.
func ImportTikTok(opts TikTokImportOptions) ([]corpus.RawMessage, error) {
	dir := opts.File

	if strings.HasSuffix(opts.File, ".zip") {
		dir = "/tmp/replicateme-tiktok-export"
		if err := extractZip(opts.File, dir); err != nil {
			return nil, fmt.Errorf("extract tiktok zip: %w", err)
		}
	}

	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("TikTok export not found at %s", dir)
	}

	var messages []corpus.RawMessage

	// import comments from various known paths
	commentsPaths := []string{
		filepath.Join("Activity", "Comments", "comments.json"),
		filepath.Join("Comment", "Comments.json"),
		filepath.Join("Activity", "Comment", "comments.json"),
		"comments.json",
	}

	for _, cp := range commentsPaths {
		commentsFile := filepath.Join(dir, cp)
		if _, err := os.Stat(commentsFile); err != nil {
			continue
		}

		data, err := os.ReadFile(commentsFile)
		if err != nil {
			continue
		}

		var cl tikTokCommentList
		if json.Unmarshal(data, &cl) != nil {
			continue
		}

		for i, c := range cl.CommentList {
			if c.Comment == "" {
				continue
			}

			ts, err := parseTikTokDate(c.Date)
			if err != nil {
				continue
			}
			if opts.Since != nil && ts.Before(*opts.Since) {
				continue
			}

			messages = append(messages, corpus.RawMessage{
				ID:         fmt.Sprintf("tiktok-comment-%d", i),
				Text:       c.Comment,
				Platform:   corpus.TikTok,
				Timestamp:  ts,
				IsFromUser: true,
				Metadata:   map[string]any{"type": "comment"},
			})
		}
		break // found and parsed a comments file
	}

	// import DMs from various known paths
	dmDirs := []string{
		filepath.Join(dir, "Activity", "Direct Messages"),
		filepath.Join(dir, "Direct Messages"),
		filepath.Join(dir, "Activity", "Messages"),
	}

	for _, dmDir := range dmDirs {
		if _, err := os.Stat(dmDir); err != nil {
			continue
		}

		entries, err := os.ReadDir(dmDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			var dmPath string
			if entry.IsDir() {
				// look for JSON files inside subdirectory
				subFiles, _ := os.ReadDir(filepath.Join(dmDir, entry.Name()))
				for _, sf := range subFiles {
					if strings.HasSuffix(sf.Name(), ".json") {
						dmPath = filepath.Join(dmDir, entry.Name(), sf.Name())
						break
					}
				}
			} else if strings.HasSuffix(entry.Name(), ".json") {
				dmPath = filepath.Join(dmDir, entry.Name())
			}

			if dmPath == "" {
				continue
			}

			data, err := os.ReadFile(dmPath)
			if err != nil {
				continue
			}

			var conv tikTokDMConversation
			if json.Unmarshal(data, &conv) != nil {
				continue
			}

			for j, msg := range conv.Messages {
				if msg.Content == "" {
					continue
				}

				// filter by username if provided
				if opts.Username != "" && msg.From != opts.Username {
					continue
				}

				ts, err := parseTikTokDate(msg.Date)
				if err != nil {
					continue
				}
				if opts.Since != nil && ts.Before(*opts.Since) {
					continue
				}

				messages = append(messages, corpus.RawMessage{
					ID:         fmt.Sprintf("tiktok-dm-%s-%d", entry.Name(), j),
					Text:       msg.Content,
					Platform:   corpus.TikTok,
					Timestamp:  ts,
					IsFromUser: true,
					Metadata: map[string]any{
						"type":         "dm",
						"conversation": entry.Name(),
					},
				})
			}
		}
		break // found a DM directory
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}

func parseTikTokDate(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, dateStr); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable TikTok date: %s", dateStr)
}
