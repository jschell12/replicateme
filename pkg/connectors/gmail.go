package connectors

import (
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

// GmailImportOptions controls Gmail/mbox import.
type GmailImportOptions struct {
	File  string // path to .mbox file, ZIP, or extracted Takeout directory
	Email string // user's email to identify sent messages
	Since *time.Time
}

// ImportGmail reads sent messages from a Gmail mbox export.
func ImportGmail(opts GmailImportOptions) ([]corpus.RawMessage, error) {
	mboxPath := opts.File

	if strings.HasSuffix(opts.File, ".zip") {
		dir := "/tmp/replicateme-gmail-export"
		if err := extractZip(opts.File, dir); err != nil {
			return nil, fmt.Errorf("extract gmail zip: %w", err)
		}
		// find .mbox files
		var found []string
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".mbox") {
				found = append(found, path)
			}
			return nil
		})
		if len(found) == 0 {
			return nil, fmt.Errorf("no .mbox files found in archive")
		}
		// prefer Sent Mail, then All mail, then first
		mboxPath = found[0]
		for _, f := range found {
			lower := strings.ToLower(f)
			if strings.Contains(lower, "sent") {
				mboxPath = f
				break
			}
			if strings.Contains(lower, "all") && strings.Contains(lower, "mail") {
				mboxPath = f
			}
		}
	}

	if _, err := os.Stat(mboxPath); err != nil {
		return nil, fmt.Errorf("mbox file not found at %s", mboxPath)
	}

	content, err := os.ReadFile(mboxPath)
	if err != nil {
		return nil, err
	}

	return parseMbox(string(content), opts)
}

func parseMbox(content string, opts GmailImportOptions) ([]corpus.RawMessage, error) {
	var messages []corpus.RawMessage

	// split on mbox "From " delimiter
	parts := regexp.MustCompile(`(?m)^From `).Split(content, -1)

	for i, raw := range parts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		// split headers from body at first blank line
		headerEnd := strings.Index(raw, "\n\n")
		if headerEnd == -1 {
			continue
		}

		headerBlock := raw[:headerEnd]
		body := raw[headerEnd+2:]

		headers := parseHeaders(headerBlock)

		from := headers["from"]
		dateStr := headers["date"]
		subject := headers["subject"]

		// only keep sent messages
		isSent := false
		if opts.Email != "" {
			isSent = strings.Contains(strings.ToLower(from), strings.ToLower(opts.Email))
		} else {
			labels := headers["x-gmail-labels"]
			isSent = strings.Contains(strings.ToLower(labels), "sent")
		}
		if !isSent {
			continue
		}

		ts, err := parseEmailDate(dateStr)
		if err != nil {
			continue
		}
		if opts.Since != nil && ts.Before(*opts.Since) {
			continue
		}

		plainText := extractPlainText(body)
		if len(plainText) < 2 {
			continue
		}

		messages = append(messages, corpus.RawMessage{
			ID:         fmt.Sprintf("gmail-%d", i),
			Text:       plainText,
			Platform:   corpus.Email,
			Timestamp:  ts,
			IsFromUser: true,
			Metadata: map[string]any{
				"subject": subject,
				"to":      headers["to"],
			},
		})
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages, nil
}

func parseHeaders(block string) map[string]string {
	headers := make(map[string]string)
	// unfold continuation lines
	unfolded := regexp.MustCompile(`\n[ \t]+`).ReplaceAllString(block, " ")
	for _, line := range strings.Split(unfolded, "\n") {
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(line[:idx]))
		value := strings.TrimSpace(line[idx+1:])
		headers[key] = value
	}
	return headers
}

func parseEmailDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	// try common email date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
		"2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}
	// strip trailing timezone name in parens: " (PST)"
	cleaned := regexp.MustCompile(`\s*\([^)]*\)\s*$`).ReplaceAllString(dateStr, "")
	for _, f := range formats {
		if t, err := time.Parse(f, cleaned); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable date: %s", dateStr)
}

func extractPlainText(body string) string {
	text := body

	// if multipart, try to extract text/plain part
	if strings.Contains(text, "Content-Type: text/plain") {
		parts := regexp.MustCompile(`--[^\n]+`).Split(text, -1)
		for _, p := range parts {
			if strings.Contains(p, "text/plain") {
				bodyStart := strings.Index(p, "\n\n")
				if bodyStart != -1 {
					text = p[bodyStart+2:]
					break
				}
			}
		}
	}

	// strip quoted replies
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, ">") {
			lines = append(lines, line)
		}
	}
	text = strings.Join(lines, "\n")

	// strip "On ... wrote:" blocks
	text = regexp.MustCompile(`(?m)On .+wrote:\s*$`).ReplaceAllString(text, "")

	// strip signatures
	if idx := strings.Index(text, "\n-- \n"); idx != -1 {
		text = text[:idx]
	}

	// decode quoted-printable soft line breaks
	text = regexp.MustCompile(`=\r?\n`).ReplaceAllString(text, "")

	// decode common QP entities
	qpRe := regexp.MustCompile(`=([0-9A-Fa-f]{2})`)
	text = qpRe.ReplaceAllStringFunc(text, func(match string) string {
		hex := match[1:]
		val, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return match
		}
		return string(rune(val))
	})

	return strings.TrimSpace(text)
}
