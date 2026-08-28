package connectors

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jschell12/replicateme/pkg/corpus"
	_ "modernc.org/sqlite"
)

const appleEpochOffset = 978307200

// IMessageOptions controls iMessage import behavior.
type IMessageOptions struct {
	Since     *time.Time
	DBPath    string // override ~/Library/Messages/chat.db
	CopyToTmp bool   // default true; copies the db to /tmp to work around FDA
}

// ImportIMessages reads sent messages from the local iMessage database.
func ImportIMessages(opts IMessageOptions) ([]corpus.RawMessage, error) {
	source := opts.DBPath
	if source == "" {
		home, _ := os.UserHomeDir()
		source = filepath.Join(home, "Library", "Messages", "chat.db")
	}

	if _, err := os.Stat(source); err != nil {
		return nil, fmt.Errorf("iMessage database not found at %s", source)
	}

	dbPath := source
	if !opts.CopyToTmp || opts.DBPath == "" {
		// default: copy to tmp for FDA workaround
		dbPath = "/tmp/replicateme-chat.db"
		if err := copyFile(source, dbPath); err != nil {
			return nil, fmt.Errorf("copy chat.db to tmp: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			m.ROWID,
			m.text,
			m.date,
			m.is_from_me,
			h.id,
			m.cache_roomnames,
			m.service,
			m.thread_originator_guid
		FROM message m
		LEFT JOIN handle h ON m.handle_id = h.ROWID
		WHERE m.is_from_me = 1
			AND m.text IS NOT NULL
			AND length(m.text) > 0
			AND m.associated_message_type = 0
	`
	args := []any{}

	if opts.Since != nil {
		appleTS := (opts.Since.Unix() - appleEpochOffset) * 1_000_000_000
		query += " AND m.date >= ?"
		args = append(args, appleTS)
	}

	query += " ORDER BY m.date ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []corpus.RawMessage
	for rows.Next() {
		var (
			rowid          int64
			text           string
			date           int64
			isFromMe       int
			handleID       sql.NullString
			cacheRoomnames sql.NullString
			service        string
			threadGUID     sql.NullString
		)
		if err := rows.Scan(&rowid, &text, &date, &isFromMe, &handleID, &cacheRoomnames, &service, &threadGUID); err != nil {
			return nil, err
		}

		ts := time.Unix(date/1_000_000_000+appleEpochOffset, 0).UTC()

		meta := map[string]any{
			"isGroupChat": cacheRoomnames.Valid,
			"service":     service,
		}
		if handleID.Valid {
			meta["recipient"] = handleID.String
		}
		if threadGUID.Valid {
			meta["threadId"] = threadGUID.String
		}

		messages = append(messages, corpus.RawMessage{
			ID:         fmt.Sprintf("imessage-%d", rowid),
			Text:       text,
			Platform:   corpus.IMMessage,
			Timestamp:  ts,
			IsFromUser: true,
			Metadata:   meta,
		})
	}

	return messages, rows.Err()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
