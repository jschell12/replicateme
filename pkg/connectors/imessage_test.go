package connectors

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupIMessageDB(t *testing.T, dir string) (string, *sql.DB) {
	t.Helper()
	dbPath := filepath.Join(dir, "chat.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// create the schema matching what ImportIMessages expects
	stmts := []string{
		`CREATE TABLE handle (
			ROWID INTEGER PRIMARY KEY,
			id TEXT
		)`,
		`CREATE TABLE message (
			ROWID INTEGER PRIMARY KEY,
			text TEXT,
			date INTEGER,
			is_from_me INTEGER DEFAULT 0,
			handle_id INTEGER,
			cache_roomnames TEXT,
			service TEXT DEFAULT 'iMessage',
			thread_originator_guid TEXT,
			associated_message_type INTEGER DEFAULT 0
		)`,
		`CREATE TABLE chat_handle_join (
			chat_id INTEGER,
			handle_id INTEGER
		)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	return dbPath, db
}

func insertHandle(t *testing.T, db *sql.DB, rowid int64, id string) {
	t.Helper()
	if _, err := db.Exec("INSERT INTO handle (ROWID, id) VALUES (?, ?)", rowid, id); err != nil {
		t.Fatalf("insert handle: %v", err)
	}
}

func insertMessage(t *testing.T, db *sql.DB, rowid int64, text string, ts time.Time, isFromMe int, handleID int64, roomName, service string) {
	t.Helper()
	// Convert to Apple epoch nanoseconds
	appleTS := (ts.Unix() - appleEpochOffset) * 1_000_000_000

	var roomNamePtr *string
	if roomName != "" {
		roomNamePtr = &roomName
	}

	if _, err := db.Exec(
		"INSERT INTO message (ROWID, text, date, is_from_me, handle_id, cache_roomnames, service, thread_originator_guid, associated_message_type) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, 0)",
		rowid, text, appleTS, isFromMe, handleID, roomNamePtr, service,
	); err != nil {
		t.Fatalf("insert message: %v", err)
	}
}

func TestImportIMessages_BasicMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "Hello from me", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	insertMessage(t, db, 2, "Hello back", time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC), 0, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{
		DBPath:    dbPath,
		CopyToTmp: true, // set true so it uses DBPath directly per the logic
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// only is_from_me=1 should be returned
	if len(result) != 1 {
		t.Fatalf("expected 1 message (from me only), got %d", len(result))
	}
	if result[0].Text != "Hello from me" {
		t.Errorf("expected 'Hello from me', got %q", result[0].Text)
	}
	if result[0].ID != "imessage-1" {
		t.Errorf("expected ID 'imessage-1', got %q", result[0].ID)
	}
}

func TestImportIMessages_AppleEpochConversion(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	// 2023-11-14 15:00:00 UTC
	expected := time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC)
	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "Test timestamp", expected, 1, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if !result[0].Timestamp.Equal(expected) {
		t.Errorf("expected timestamp %v, got %v", expected, result[0].Timestamp)
	}
}

func TestImportIMessages_SinceFilter(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "Old message", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	insertMessage(t, db, 2, "New message", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	db.Close()

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ImportIMessages(IMessageOptions{
		DBPath:    dbPath,
		CopyToTmp: true,
		Since:     &since,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "New message" {
		t.Errorf("expected 'New message', got %q", result[0].Text)
	}
}

func TestImportIMessages_GroupChatMetadata(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "Group msg", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "chat123", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Metadata["isGroupChat"] != true {
		t.Errorf("expected isGroupChat=true, got %v", result[0].Metadata["isGroupChat"])
	}
}

func TestImportIMessages_NonGroupChat(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "Direct msg", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Metadata["isGroupChat"] != false {
		t.Errorf("expected isGroupChat=false, got %v", result[0].Metadata["isGroupChat"])
	}
}

func TestImportIMessages_RecipientMetadata(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "With recipient", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Metadata["recipient"] != "+15551234567" {
		t.Errorf("expected recipient '+15551234567', got %v", result[0].Metadata["recipient"])
	}
	if result[0].Metadata["service"] != "iMessage" {
		t.Errorf("expected service 'iMessage', got %v", result[0].Metadata["service"])
	}
}

func TestImportIMessages_SkipsNullText(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	// message with NULL text (SQL default)
	db.Exec("INSERT INTO message (ROWID, text, date, is_from_me, handle_id, service, associated_message_type) VALUES (1, NULL, ?, 1, 1, 'iMessage', 0)",
		(time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).Unix()-appleEpochOffset)*1_000_000_000)
	// message with empty text
	db.Exec("INSERT INTO message (ROWID, text, date, is_from_me, handle_id, service, associated_message_type) VALUES (2, '', ?, 1, 1, 'iMessage', 0)",
		(time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC).Unix()-appleEpochOffset)*1_000_000_000)
	// valid message
	insertMessage(t, db, 3, "Real message", time.Date(2023, 11, 14, 17, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message (null/empty skipped), got %d", len(result))
	}
}

func TestImportIMessages_SkipsAssociatedMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	// associated message (like a reaction) with associated_message_type != 0
	db.Exec("INSERT INTO message (ROWID, text, date, is_from_me, handle_id, service, associated_message_type) VALUES (1, 'Loved an image', ?, 1, 1, 'iMessage', 2000)",
		(time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).Unix()-appleEpochOffset)*1_000_000_000)
	// normal message
	insertMessage(t, db, 2, "Normal msg", time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message (associated skipped), got %d", len(result))
	}
}

func TestImportIMessages_MultipleMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertHandle(t, db, 2, "+15559876543")

	insertMessage(t, db, 1, "First msg", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	insertMessage(t, db, 2, "Second msg", time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC), 1, 2, "", "SMS")
	insertMessage(t, db, 3, "Third msg", time.Date(2023, 11, 14, 17, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	// should be ordered by date ASC
	if result[0].Text != "First msg" {
		t.Errorf("expected 'First msg' first, got %q", result[0].Text)
	}
	if result[2].Text != "Third msg" {
		t.Errorf("expected 'Third msg' last, got %q", result[2].Text)
	}
}

func TestImportIMessages_NonexistentDB(t *testing.T) {
	_, err := ImportIMessages(IMessageOptions{
		DBPath:    "/nonexistent/chat.db",
		CopyToTmp: true,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent database")
	}
}

func TestImportIMessages_EmptyDB(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(result))
	}
}

func TestImportIMessages_SMSService(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "SMS message", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "", "SMS")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Metadata["service"] != "SMS" {
		t.Errorf("expected service 'SMS', got %v", result[0].Metadata["service"])
	}
}

func TestImportIMessages_PlatformIsIMMessage(t *testing.T) {
	dir := t.TempDir()
	dbPath, db := setupIMessageDB(t, dir)

	insertHandle(t, db, 1, "+15551234567")
	insertMessage(t, db, 1, "Check platform", time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC), 1, 1, "", "iMessage")
	db.Close()

	result, err := ImportIMessages(IMessageOptions{DBPath: dbPath, CopyToTmp: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if string(result[0].Platform) != "imessage" {
		t.Errorf("expected platform 'imessage', got %q", result[0].Platform)
	}
}

// TestCopyFile tests the copyFile helper
func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")

	os.WriteFile(src, []byte("test content"), 0o644)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("expected 'test content', got %q", string(data))
	}
}
