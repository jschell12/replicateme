package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jschell12/replicateme/pkg/corpus"
)

// TwitterImportOptions controls Twitter/X archive import.
type TwitterImportOptions struct {
	File       string // path to ZIP or extracted archive directory
	Since      *time.Time
	IncludeDMs *bool // default true
}

type tweetEntry struct {
	Tweet struct {
		IDStr                  string `json:"id_str"`
		FullText               string `json:"full_text"`
		CreatedAt              string `json:"created_at"`
		InReplyToStatusIDStr   string `json:"in_reply_to_status_id_str,omitempty"`
		InReplyToScreenName    string `json:"in_reply_to_screen_name,omitempty"`
	} `json:"tweet"`
}

type dmConversationWrapper struct {
	DMConversation struct {
		Messages []struct {
			MessageCreate *struct {
				ID          string `json:"id"`
				Text        string `json:"text"`
				CreatedAt   string `json:"createdAt"`
				SenderID    string `json:"senderId"`
				RecipientID string `json:"recipientId"`
			} `json:"messageCreate,omitempty"`
		} `json:"messages"`
	} `json:"dmConversation"`
}

// ImportTwitter reads tweets and DMs from a Twitter/X data archive.
func ImportTwitter(opts TwitterImportOptions) ([]corpus.RawMessage, error) {
	dir := opts.File

	if strings.HasSuffix(opts.File, ".zip") {
		dir = "/tmp/replicateme-twitter-export"
		if err := extractZip(opts.File, dir); err != nil {
			return nil, fmt.Errorf("extract twitter zip: %w", err)
		}
	}

	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("Twitter archive not found at %s", dir)
	}

	dataDir := dir
	if _, err := os.Stat(filepath.Join(dir, "data")); err == nil {
		dataDir = filepath.Join(dir, "data")
	}

	var messages []corpus.RawMessage

	// import tweets
	tweetsFile := findFile(dataDir, []string{"tweets.js", "tweet.js"})
	if tweetsFile != "" {
		var tweets []tweetEntry
		if err := parseTwitterJS(tweetsFile, &tweets); err == nil {
			for _, entry := range tweets {
				t := entry.Tweet
				if t.FullText == "" {
					continue
				}
				ts, err := time.Parse("Mon Jan 02 15:04:05 -0700 2006", t.CreatedAt)
				if err != nil {
					continue
				}
				if opts.Since != nil && ts.Before(*opts.Since) {
					continue
				}

				messages = append(messages, corpus.RawMessage{
					ID:         fmt.Sprintf("twitter-%s", t.IDStr),
					Text:       cleanTweetText(t.FullText),
					Platform:   corpus.Twitter,
					Timestamp:  ts.UTC(),
					IsFromUser: true,
					Metadata: map[string]any{
						"isReply": t.InReplyToStatusIDStr != "",
						"replyTo": t.InReplyToScreenName,
					},
				})
			}
		}
	}

	// import DMs
	includeDMs := opts.IncludeDMs == nil || *opts.IncludeDMs
	if includeDMs {
		dmsFile := findFile(dataDir, []string{"direct-messages.js", "direct_messages.js"})
		if dmsFile != "" {
			var dms []dmConversationWrapper
			if err := parseTwitterJS(dmsFile, &dms); err == nil {
				for _, conv := range dms {
					for _, msg := range conv.DMConversation.Messages {
						dm := msg.MessageCreate
						if dm == nil || dm.Text == "" {
							continue
						}
						createdAtMs, err := strconv.ParseInt(dm.CreatedAt, 10, 64)
						if err != nil {
							continue
						}
						ts := time.UnixMilli(createdAtMs).UTC()
						if opts.Since != nil && ts.Before(*opts.Since) {
							continue
						}

						messages = append(messages, corpus.RawMessage{
							ID:         fmt.Sprintf("twitter-dm-%s", dm.ID),
							Text:       dm.Text,
							Platform:   corpus.Twitter,
							Timestamp:  ts,
							IsFromUser: true,
							Metadata: map[string]any{
								"isDM":        true,
								"senderId":    dm.SenderID,
								"recipientId": dm.RecipientID,
							},
						})
					}
				}
			}
		}
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}

func parseTwitterJS(filePath string, target any) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	s := string(content)
	// strip "window.YTD.xxx.part0 = " prefix
	if idx := strings.Index(s, "="); idx != -1 {
		s = strings.TrimSpace(s[idx+1:])
	}
	// strip trailing semicolon
	s = strings.TrimRight(s, ";")
	return json.Unmarshal([]byte(s), target)
}

func findFile(dir string, names []string) string {
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

var tcoURLRe = regexp.MustCompile(`https?://t\.co/\w+`)

func cleanTweetText(text string) string {
	cleaned := tcoURLRe.ReplaceAllString(text, "")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", `"`)
	return cleaned
}
