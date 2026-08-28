package connectors

import (
	"archive/zip"
	"encoding/json"
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

// SlackImportOptions controls Slack workspace export import.
type SlackImportOptions struct {
	File     string // path to ZIP or extracted directory
	UserID   string // Slack user ID to filter
	UserName string // display name to resolve via users.json
	Since    *time.Time
}

type slackMessage struct {
	Type     string `json:"type"`
	Subtype  string `json:"subtype,omitempty"`
	User     string `json:"user,omitempty"`
	Text     string `json:"text,omitempty"`
	TS       string `json:"ts,omitempty"`
	ThreadTS string `json:"thread_ts,omitempty"`
}

type slackUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name,omitempty"`
	Profile  struct {
		DisplayName string `json:"display_name,omitempty"`
	} `json:"profile,omitempty"`
}

// ImportSlack reads messages from a Slack workspace export.
func ImportSlack(opts SlackImportOptions) ([]corpus.RawMessage, error) {
	dir := opts.File

	if strings.HasSuffix(opts.File, ".zip") {
		dir = "/tmp/replicateme-slack-export"
		if err := extractZip(opts.File, dir); err != nil {
			return nil, fmt.Errorf("extract slack zip: %w", err)
		}
	}

	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("Slack export not found at %s", dir)
	}

	// resolve user ID from users.json if needed
	userID := opts.UserID
	if userID == "" && opts.UserName != "" {
		usersFile := filepath.Join(dir, "users.json")
		if data, err := os.ReadFile(usersFile); err == nil {
			var users []slackUser
			if json.Unmarshal(data, &users) == nil {
				for _, u := range users {
					if u.Name == opts.UserName || u.RealName == opts.UserName || u.Profile.DisplayName == opts.UserName {
						userID = u.ID
						break
					}
				}
			}
		}
	}

	var messages []corpus.RawMessage

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		channel := entry.Name()
		channelDir := filepath.Join(dir, channel)

		files, err := os.ReadDir(channelDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if filepath.Ext(f.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(channelDir, f.Name()))
			if err != nil {
				continue
			}
			var msgs []slackMessage
			if json.Unmarshal(data, &msgs) != nil {
				continue
			}

			for _, msg := range msgs {
				if msg.Type != "message" || msg.Subtype != "" {
					continue
				}
				if msg.Text == "" || msg.TS == "" {
					continue
				}
				if userID != "" && msg.User != userID {
					continue
				}

				var tsFloat float64
				fmt.Sscanf(msg.TS, "%f", &tsFloat)
				ts := time.Unix(int64(tsFloat), int64((tsFloat-float64(int64(tsFloat)))*1e9)).UTC()

				if opts.Since != nil && ts.Before(*opts.Since) {
					continue
				}

				messages = append(messages, corpus.RawMessage{
					ID:         fmt.Sprintf("slack-%s", msg.TS),
					Text:       cleanSlackText(msg.Text),
					Platform:   corpus.Slack,
					Timestamp:  ts,
					IsFromUser: true,
					Metadata: map[string]any{
						"channel":  channel,
						"threadTs": msg.ThreadTS,
						"userId":   msg.User,
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

var (
	slackUserMentionRe   = regexp.MustCompile(`<@[A-Z0-9]+>`)
	slackChannelRefRe    = regexp.MustCompile(`<#[A-Z0-9]+\|([^>]+)>`)
	slackURLWithLabelRe  = regexp.MustCompile(`<(https?://[^|>]+)\|([^>]+)>`)
	slackURLBareRe       = regexp.MustCompile(`<(https?://[^>]+)>`)
)

func cleanSlackText(text string) string {
	cleaned := slackUserMentionRe.ReplaceAllString(text, "@user")
	cleaned = slackChannelRefRe.ReplaceAllString(cleaned, "#$1")
	cleaned = slackURLWithLabelRe.ReplaceAllString(cleaned, "$2")
	cleaned = slackURLBareRe.ReplaceAllString(cleaned, "$1")
	return cleaned
}

// extractZip extracts a ZIP archive to dst, clearing dst first.
func extractZip(src, dst string) error {
	os.RemoveAll(dst)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(path, filepath.Clean(dst)+string(os.PathSeparator)) {
			continue // zip slip protection
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
