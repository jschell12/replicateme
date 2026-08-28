package corpus

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	dbOnce sync.Once
	dbInst *sql.DB
)

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".replicateme")
}

func dbPath() string {
	return filepath.Join(configDir(), "corpus.db")
}

// DB returns the singleton database connection, creating and migrating the
// schema on first call.
func DB() (*sql.DB, error) {
	var initErr error
	dbOnce.Do(func() {
		dir := configDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			initErr = fmt.Errorf("create config dir: %w", err)
			return
		}
		db, err := sql.Open("sqlite", dbPath())
		if err != nil {
			initErr = fmt.Errorf("open db: %w", err)
			return
		}
		if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
			initErr = fmt.Errorf("set WAL: %w", err)
			return
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			initErr = fmt.Errorf("set FK: %w", err)
			return
		}
		if err := migrate(db); err != nil {
			initErr = fmt.Errorf("migrate: %w", err)
			return
		}
		dbInst = db
	})
	if initErr != nil {
		return nil, initErr
	}
	return dbInst, nil
}

func migrate(db *sql.DB) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		text TEXT NOT NULL,
		platform TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		is_from_user INTEGER NOT NULL DEFAULT 1,
		metadata TEXT,
		ingested_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_messages_platform ON messages(platform);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);

	CREATE TABLE IF NOT EXISTS profiles (
		platform TEXT PRIMARY KEY,
		profile TEXT NOT NULL,
		message_count INTEGER NOT NULL,
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS ingest_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		message_count INTEGER NOT NULL,
		ingested_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`
	_, err := db.Exec(ddl)
	return err
}

// StoreResult reports how many messages were inserted vs skipped (duplicate).
type StoreResult struct {
	Inserted int
	Skipped  int
}

// StoreMessages inserts messages into the corpus, skipping duplicates by ID.
func StoreMessages(messages []RawMessage) (StoreResult, error) {
	db, err := DB()
	if err != nil {
		return StoreResult{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return StoreResult{}, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO messages (id, text, platform, timestamp, is_from_user, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return StoreResult{}, err
	}
	defer stmt.Close()

	var res StoreResult
	for _, msg := range messages {
		var metaJSON *string
		if msg.Metadata != nil {
			b, _ := json.Marshal(msg.Metadata)
			s := string(b)
			metaJSON = &s
		}
		isFrom := 0
		if msg.IsFromUser {
			isFrom = 1
		}
		r, err := stmt.Exec(msg.ID, msg.Text, string(msg.Platform),
			msg.Timestamp.UTC().Format(time.RFC3339), isFrom, metaJSON)
		if err != nil {
			return StoreResult{}, err
		}
		affected, _ := r.RowsAffected()
		if affected > 0 {
			res.Inserted++
		} else {
			res.Skipped++
		}
	}

	if err := tx.Commit(); err != nil {
		return StoreResult{}, err
	}
	return res, nil
}

// GetMessagesOpts controls filtering for GetMessages.
type GetMessagesOpts struct {
	Platform Platform
	Limit    int
	Since    *time.Time
	Random   bool
}

// GetMessages retrieves user messages from the corpus with optional filters.
func GetMessages(opts GetMessagesOpts) ([]RawMessage, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}

	query := "SELECT id, text, platform, timestamp, is_from_user, metadata FROM messages WHERE is_from_user = 1"
	args := []any{}

	if opts.Platform != "" {
		query += " AND platform = ?"
		args = append(args, string(opts.Platform))
	}
	if opts.Since != nil {
		query += " AND timestamp >= ?"
		args = append(args, opts.Since.UTC().Format(time.RFC3339))
	}

	if opts.Random {
		query += " ORDER BY RANDOM()"
	} else {
		query += " ORDER BY timestamp DESC"
	}

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

func scanMessages(rows *sql.Rows) ([]RawMessage, error) {
	var out []RawMessage
	for rows.Next() {
		var (
			msg      RawMessage
			plat     string
			ts       string
			isFrom   int
			metaJSON sql.NullString
		)
		if err := rows.Scan(&msg.ID, &msg.Text, &plat, &ts, &isFrom, &metaJSON); err != nil {
			return nil, err
		}
		msg.Platform = Platform(plat)
		msg.Timestamp, _ = time.Parse(time.RFC3339, ts)
		msg.IsFromUser = isFrom == 1
		if metaJSON.Valid {
			_ = json.Unmarshal([]byte(metaJSON.String), &msg.Metadata)
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

// GetExamples returns example messages, preferring the given platform and
// supplementing with others if needed.
func GetExamples(platform Platform, count int) ([]RawMessage, error) {
	if count <= 0 {
		count = 20
	}
	if platform != "" {
		msgs, err := GetMessages(GetMessagesOpts{Platform: platform, Limit: count, Random: true})
		if err != nil {
			return nil, err
		}
		if len(msgs) >= count {
			return msgs, nil
		}
		remaining := count - len(msgs)
		others, err := GetMessages(GetMessagesOpts{Limit: remaining, Random: true})
		if err != nil {
			return nil, err
		}
		return append(msgs, others...), nil
	}
	return GetMessages(GetMessagesOpts{Limit: count, Random: true})
}

// SaveProfile upserts a style profile for a platform (or "combined").
func SaveProfile(platform string, profile StyleProfile, messageCount int) error {
	db, err := DB()
	if err != nil {
		return err
	}
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT OR REPLACE INTO profiles (platform, profile, message_count, updated_at)
		VALUES (?, ?, ?, datetime('now'))
	`, platform, string(profileJSON), messageCount)
	return err
}

// ProfileResult is a stored profile with its message count.
type ProfileResult struct {
	Profile      StyleProfile
	MessageCount int
}

// GetProfile retrieves a single platform profile (or "combined").
func GetProfile(platform string) (*ProfileResult, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	row := db.QueryRow("SELECT profile, message_count FROM profiles WHERE platform = ?", platform)
	var profileJSON string
	var count int
	if err := row.Scan(&profileJSON, &count); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var prof StyleProfile
	if err := json.Unmarshal([]byte(profileJSON), &prof); err != nil {
		return nil, err
	}
	return &ProfileResult{Profile: prof, MessageCount: count}, nil
}

// PlatformProfile is a profile row with its platform label.
type PlatformProfile struct {
	Platform     string
	Profile      StyleProfile
	MessageCount int
}

// GetAllProfiles returns all stored style profiles.
func GetAllProfiles() ([]PlatformProfile, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query("SELECT platform, profile, message_count FROM profiles ORDER BY platform")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PlatformProfile
	for rows.Next() {
		var pp PlatformProfile
		var profileJSON string
		if err := rows.Scan(&pp.Platform, &profileJSON, &pp.MessageCount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(profileJSON), &pp.Profile); err != nil {
			return nil, err
		}
		out = append(out, pp)
	}
	return out, rows.Err()
}

// LogIngest records an ingest operation in the log table.
func LogIngest(source string, messageCount int) error {
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO ingest_log (source, message_count) VALUES (?, ?)", source, messageCount)
	return err
}

// GetCorpusStats returns aggregate corpus statistics.
func GetCorpusStats() (*CorpusStats, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&total); err != nil {
		return nil, err
	}

	platRows, err := db.Query("SELECT platform, COUNT(*) as count FROM messages GROUP BY platform ORDER BY count DESC")
	if err != nil {
		return nil, err
	}
	defer platRows.Close()

	var byPlat []PlatformCount
	for platRows.Next() {
		var pc PlatformCount
		if err := platRows.Scan(&pc.Platform, &pc.Count); err != nil {
			return nil, err
		}
		byPlat = append(byPlat, pc)
	}
	if err := platRows.Err(); err != nil {
		return nil, err
	}

	profRows, err := db.Query("SELECT platform, message_count, updated_at FROM profiles ORDER BY platform")
	if err != nil {
		return nil, err
	}
	defer profRows.Close()

	var profiles []ProfileSummary
	for profRows.Next() {
		var ps ProfileSummary
		if err := profRows.Scan(&ps.Platform, &ps.MessageCount, &ps.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, ps)
	}
	if err := profRows.Err(); err != nil {
		return nil, err
	}

	return &CorpusStats{
		TotalMessages: total,
		ByPlatform:    byPlat,
		Profiles:      profiles,
	}, nil
}

// CloseDB closes the database connection if open.
func CloseDB() error {
	if dbInst != nil {
		return dbInst.Close()
	}
	return nil
}
