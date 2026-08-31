package connectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportSlack_BasicMessages(t *testing.T) {
	dir := t.TempDir()

	// create users.json
	users := []slackUser{
		{ID: "U001", Name: "alice", RealName: "Alice Smith"},
		{ID: "U002", Name: "bob", RealName: "Bob Jones"},
	}
	writeJSON(t, filepath.Join(dir, "users.json"), users)

	// create channel directory with messages
	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "hello world", TS: "1700000000.000100"},
		{Type: "message", User: "U002", Text: "hi there", TS: "1700000001.000200"},
	}
	writeJSON(t, filepath.Join(chanDir, "2023-11-14.json"), msgs)

	result, err := ImportSlack(SlackImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Text != "hello world" {
		t.Errorf("expected 'hello world', got %q", result[0].Text)
	}
	if result[0].Metadata["channel"] != "general" {
		t.Errorf("expected channel 'general', got %v", result[0].Metadata["channel"])
	}
	// check sorted by timestamp
	if !result[0].Timestamp.Before(result[1].Timestamp) {
		t.Error("messages should be sorted by timestamp")
	}
}

func TestImportSlack_UserFilterByName(t *testing.T) {
	dir := t.TempDir()

	users := []slackUser{
		{ID: "U001", Name: "alice", RealName: "Alice Smith"},
		{ID: "U002", Name: "bob", RealName: "Bob Jones"},
	}
	writeJSON(t, filepath.Join(dir, "users.json"), users)

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "from alice", TS: "1700000000.000100"},
		{Type: "message", User: "U002", Text: "from bob", TS: "1700000001.000200"},
	}
	writeJSON(t, filepath.Join(chanDir, "2023-11-14.json"), msgs)

	result, err := ImportSlack(SlackImportOptions{File: dir, UserName: "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "from alice" {
		t.Errorf("expected 'from alice', got %q", result[0].Text)
	}
}

func TestImportSlack_UserFilterByRealName(t *testing.T) {
	dir := t.TempDir()

	users := []slackUser{
		{ID: "U001", Name: "alice", RealName: "Alice Smith"},
	}
	writeJSON(t, filepath.Join(dir, "users.json"), users)

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "from alice", TS: "1700000000.000100"},
	}
	writeJSON(t, filepath.Join(chanDir, "2023-11-14.json"), msgs)

	result, err := ImportSlack(SlackImportOptions{File: dir, UserName: "Alice Smith"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestImportSlack_UserFilterByDisplayName(t *testing.T) {
	dir := t.TempDir()

	users := []slackUser{
		{ID: "U001", Name: "alice"},
	}
	users[0].Profile.DisplayName = "AliceD"
	writeJSON(t, filepath.Join(dir, "users.json"), users)

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "from alice", TS: "1700000000.000100"},
	}
	writeJSON(t, filepath.Join(chanDir, "2023-11-14.json"), msgs)

	result, err := ImportSlack(SlackImportOptions{File: dir, UserName: "AliceD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestImportSlack_UserFilterByID(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "from alice", TS: "1700000000.000100"},
		{Type: "message", User: "U002", Text: "from bob", TS: "1700000001.000200"},
	}
	writeJSON(t, filepath.Join(chanDir, "2023-11-14.json"), msgs)

	result, err := ImportSlack(SlackImportOptions{File: dir, UserID: "U001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "from alice" {
		t.Errorf("expected 'from alice', got %q", result[0].Text)
	}
}

func TestImportSlack_CleanMarkup(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<@U123ABC> hello", "@user hello"},
		{"check <#C456|general>", "check #general"},
		{"visit <https://example.com|Example>", "visit Example"},
		{"see <https://example.com>", "see https://example.com"},
		{"plain text", "plain text"},
		{"<@U001> said <#C002|random> has <https://foo.bar|link>", "@user said #random has link"},
	}

	for _, tc := range tests {
		got := cleanSlackText(tc.input)
		if got != tc.expected {
			t.Errorf("cleanSlackText(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestImportSlack_SinceFilter(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "old message", TS: "1600000000.000100"},
		{Type: "message", User: "U001", Text: "new message", TS: "1700000000.000200"},
	}
	writeJSON(t, filepath.Join(chanDir, "msgs.json"), msgs)

	since := time.Unix(1650000000, 0)
	result, err := ImportSlack(SlackImportOptions{File: dir, Since: &since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "new message" {
		t.Errorf("expected 'new message', got %q", result[0].Text)
	}
}

func TestImportSlack_SkipsSubtypeMessages(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", Subtype: "channel_join", User: "U001", Text: "joined", TS: "1700000000.000100"},
		{Type: "message", User: "U001", Text: "real message", TS: "1700000001.000200"},
	}
	writeJSON(t, filepath.Join(chanDir, "msgs.json"), msgs)

	result, err := ImportSlack(SlackImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "real message" {
		t.Errorf("expected 'real message', got %q", result[0].Text)
	}
}

func TestImportSlack_SkipsEmptyText(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "", TS: "1700000000.000100"},
		{Type: "message", User: "U001", Text: "real message", TS: "1700000001.000200"},
	}
	writeJSON(t, filepath.Join(chanDir, "msgs.json"), msgs)

	result, err := ImportSlack(SlackImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestImportSlack_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "general")
	os.MkdirAll(chanDir, 0o755)

	// write invalid JSON
	os.WriteFile(filepath.Join(chanDir, "bad.json"), []byte("not json"), 0o644)

	result, err := ImportSlack(SlackImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 messages from invalid JSON, got %d", len(result))
	}
}

func TestImportSlack_MultipleChannels(t *testing.T) {
	dir := t.TempDir()

	for _, ch := range []string{"general", "random"} {
		chanDir := filepath.Join(dir, ch)
		os.MkdirAll(chanDir, 0o755)
		msgs := []slackMessage{
			{Type: "message", User: "U001", Text: "msg in " + ch, TS: "1700000000.000100"},
		}
		writeJSON(t, filepath.Join(chanDir, "day.json"), msgs)
	}

	result, err := ImportSlack(SlackImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
}

func TestImportSlack_ZipImport(t *testing.T) {
	dir := t.TempDir()

	// create the workspace structure in a staging dir, then zip it
	stagingDir := filepath.Join(dir, "staging")
	chanDir := filepath.Join(stagingDir, "general")
	os.MkdirAll(chanDir, 0o755)

	msgs := []slackMessage{
		{Type: "message", User: "U001", Text: "zipped msg", TS: "1700000000.000100"},
	}
	writeJSON(t, filepath.Join(chanDir, "day.json"), msgs)

	zipPath := filepath.Join(dir, "export.zip")
	createZipFromDir(t, stagingDir, zipPath)

	result, err := ImportSlack(SlackImportOptions{File: zipPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "zipped msg" {
		t.Errorf("expected 'zipped msg', got %q", result[0].Text)
	}
}

func TestImportSlack_NonexistentDir(t *testing.T) {
	_, err := ImportSlack(SlackImportOptions{File: "/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

// helper: write JSON to a file
func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
