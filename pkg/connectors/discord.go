package connectors

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jschell12/replicateme/pkg/corpus"
)

// DiscordImportOptions controls Discord data export import.
type DiscordImportOptions struct {
	File  string // path to ZIP or extracted directory
	Since *time.Time
}

// ImportDiscord reads messages from a Discord data export.
func ImportDiscord(opts DiscordImportOptions) ([]corpus.RawMessage, error) {
	dir := opts.File

	if strings.HasSuffix(opts.File, ".zip") {
		dir = "/tmp/replicateme-discord-export"
		if err := extractZip(opts.File, dir); err != nil {
			return nil, fmt.Errorf("extract discord zip: %w", err)
		}
	}

	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("Discord export not found at %s", dir)
	}

	messagesDir := dir
	if _, err := os.Stat(filepath.Join(dir, "messages")); err == nil {
		messagesDir = filepath.Join(dir, "messages")
	}

	// load channel index
	channelNames := make(map[string]string)
	indexFile := filepath.Join(messagesDir, "index.json")
	if data, err := os.ReadFile(indexFile); err == nil {
		var index map[string]*string
		if json.Unmarshal(data, &index) == nil {
			for id, name := range index {
				if name != nil {
					channelNames[id] = *name
				}
			}
		}
	}

	var messages []corpus.RawMessage

	entries, err := os.ReadDir(messagesDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		channelID := entry.Name()
		csvPath := filepath.Join(messagesDir, channelID, "messages.csv")
		if _, err := os.Stat(csvPath); err != nil {
			continue
		}

		channelName := channelNames[channelID]
		if channelName == "" {
			channelName = channelID
		}

		f, err := os.Open(csvPath)
		if err != nil {
			continue
		}

		reader := csv.NewReader(f)
		reader.LazyQuotes = true

		// skip header
		if _, err := reader.Read(); err != nil {
			f.Close()
			continue
		}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			// columns: ID, Timestamp, Contents, Attachments
			if len(record) < 3 {
				continue
			}

			id := record[0]
			timestampStr := record[1]
			contents := record[2]

			if id == "" || contents == "" || timestampStr == "" {
				continue
			}

			ts, err := time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				// try other formats
				ts, err = time.Parse("2006-01-02 15:04:05", timestampStr)
				if err != nil {
					continue
				}
			}

			if opts.Since != nil && ts.Before(*opts.Since) {
				continue
			}

			messages = append(messages, corpus.RawMessage{
				ID:         fmt.Sprintf("discord-%s", id),
				Text:       contents,
				Platform:   corpus.Discord,
				Timestamp:  ts.UTC(),
				IsFromUser: true,
				Metadata: map[string]any{
					"channel":   channelName,
					"channelId": channelID,
				},
			})
		}
		f.Close()
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}
