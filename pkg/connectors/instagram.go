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

// InstagramImportOptions controls Instagram data download import.
type InstagramImportOptions struct {
	File     string // path to ZIP or extracted directory
	Username string // your Instagram username to identify your messages
	Since    *time.Time
}

type igConversation struct {
	Participants []struct {
		Name string `json:"name"`
	} `json:"participants"`
	Messages []struct {
		SenderName  string `json:"sender_name"`
		TimestampMs int64  `json:"timestamp_ms"`
		Content     string `json:"content,omitempty"`
		Type        string `json:"type,omitempty"`
	} `json:"messages"`
}

type igPost struct {
	Title             string `json:"title,omitempty"`
	CreationTimestamp  int64  `json:"creation_timestamp,omitempty"`
	Media             []struct {
		Title             string `json:"title,omitempty"`
		CreationTimestamp  int64  `json:"creation_timestamp,omitempty"`
	} `json:"media,omitempty"`
}

type igComment struct {
	StringMapData map[string]struct {
		Value     string `json:"value,omitempty"`
		Timestamp int64  `json:"timestamp,omitempty"`
	} `json:"string_map_data,omitempty"`
}

type igCommentsWrapper struct {
	CommentsMediaComments []igComment `json:"comments_media_comments,omitempty"`
}

// ImportInstagram reads DMs, posts, and comments from an Instagram data download.
func ImportInstagram(opts InstagramImportOptions) ([]corpus.RawMessage, error) {
	dir := opts.File

	if strings.HasSuffix(opts.File, ".zip") {
		dir = "/tmp/replicateme-instagram-export"
		if err := extractZip(opts.File, dir); err != nil {
			return nil, fmt.Errorf("extract instagram zip: %w", err)
		}
	}

	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("Instagram export not found at %s", dir)
	}

	var messages []corpus.RawMessage

	// import DMs
	inboxDir := filepath.Join(dir, "messages", "inbox")
	if _, err := os.Stat(inboxDir); err == nil {
		convEntries, _ := os.ReadDir(inboxDir)
		for _, convEntry := range convEntries {
			if !convEntry.IsDir() {
				continue
			}
			convName := convEntry.Name()
			convDir := filepath.Join(inboxDir, convName)
			files, _ := os.ReadDir(convDir)

			for _, f := range files {
				name := f.Name()
				if !strings.HasPrefix(name, "message_") || !strings.HasSuffix(name, ".json") {
					continue
				}

				data, err := os.ReadFile(filepath.Join(convDir, name))
				if err != nil {
					continue
				}
				var conv igConversation
				if json.Unmarshal(data, &conv) != nil {
					continue
				}

				for _, msg := range conv.Messages {
					if msg.Content == "" {
						continue
					}
					if opts.Username != "" && decodeIGText(msg.SenderName) != opts.Username {
						continue
					}

					ts := time.UnixMilli(msg.TimestampMs).UTC()
					if opts.Since != nil && ts.Before(*opts.Since) {
						continue
					}

					messages = append(messages, corpus.RawMessage{
						ID:         fmt.Sprintf("instagram-dm-%d", msg.TimestampMs),
						Text:       decodeIGText(msg.Content),
						Platform:   corpus.Instagram,
						Timestamp:  ts,
						IsFromUser: true,
						Metadata: map[string]any{
							"type":         "dm",
							"conversation": convName,
						},
					})
				}
			}
		}
	}

	// import post captions
	postsFile := findIGFile(dir, []string{
		filepath.Join("content", "posts_1.json"),
		filepath.Join("content", "posts.json"),
		"posts_1.json",
	})
	if postsFile != "" {
		data, err := os.ReadFile(postsFile)
		if err == nil {
			var posts []igPost
			if json.Unmarshal(data, &posts) == nil {
				for i, post := range posts {
					caption := post.Title
					if caption == "" && len(post.Media) > 0 {
						caption = post.Media[0].Title
					}
					if caption == "" {
						continue
					}

					ts := post.CreationTimestamp
					if ts == 0 && len(post.Media) > 0 {
						ts = post.Media[0].CreationTimestamp
					}
					if ts == 0 {
						continue
					}

					timestamp := time.Unix(ts, 0).UTC()
					if opts.Since != nil && timestamp.Before(*opts.Since) {
						continue
					}

					messages = append(messages, corpus.RawMessage{
						ID:         fmt.Sprintf("instagram-post-%d", i),
						Text:       decodeIGText(caption),
						Platform:   corpus.Instagram,
						Timestamp:  timestamp,
						IsFromUser: true,
						Metadata:   map[string]any{"type": "post"},
					})
				}
			}
		}
	}

	// import comments
	commentsFile := findIGFile(dir, []string{
		filepath.Join("comments", "post_comments_1.json"),
		filepath.Join("comments", "post_comments.json"),
		"comments_1.json",
	})
	if commentsFile != "" {
		data, err := os.ReadFile(commentsFile)
		if err == nil {
			var comments []igComment
			// try direct array
			if json.Unmarshal(data, &comments) != nil {
				// try wrapper object
				var wrapper igCommentsWrapper
				if json.Unmarshal(data, &wrapper) == nil {
					comments = wrapper.CommentsMediaComments
				}
			}

			for i, c := range comments {
				commentData := c.StringMapData["Comment"]
				text := commentData.Value
				ts := commentData.Timestamp
				if text == "" || ts == 0 {
					continue
				}

				timestamp := time.Unix(ts, 0).UTC()
				if opts.Since != nil && timestamp.Before(*opts.Since) {
					continue
				}

				meta := map[string]any{"type": "comment"}
				if owner, ok := c.StringMapData["Media Owner"]; ok && owner.Value != "" {
					meta["mediaOwner"] = owner.Value
				}

				messages = append(messages, corpus.RawMessage{
					ID:         fmt.Sprintf("instagram-comment-%d", i),
					Text:       decodeIGText(text),
					Platform:   corpus.Instagram,
					Timestamp:  timestamp,
					IsFromUser: true,
					Metadata:   meta,
				})
			}
		}
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}

func findIGFile(dir string, candidates []string) string {
	for _, c := range candidates {
		path := filepath.Join(dir, c)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// decodeIGText handles Instagram's encoding of non-ASCII characters:
// UTF-8 bytes encoded as escaped sequences in latin1.
var igEscapeRe = regexp.MustCompile(`\\u00([0-9a-fA-F]{2})`)

func decodeIGText(text string) string {
	// replace \u00XX escapes with the actual byte values
	decoded := igEscapeRe.ReplaceAllStringFunc(text, func(match string) string {
		hex := match[4:] // skip \u00
		val, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return match
		}
		return string(rune(val))
	})
	// The bytes are UTF-8 encoded but stored as latin1 codepoints.
	// Convert: treat each character as a byte, then interpret as UTF-8.
	buf := make([]byte, len(decoded))
	for i := 0; i < len(decoded); i++ {
		buf[i] = decoded[i]
	}
	// If it's valid UTF-8 after conversion, use it; otherwise keep original.
	result := string(buf)
	return result
}
