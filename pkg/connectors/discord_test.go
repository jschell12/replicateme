package connectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportDiscord_BasicMessages(t *testing.T) {
	dir := t.TempDir()

	// create index.json
	index := map[string]*string{}
	general := "general"
	index["c001"] = &general
	writeJSON(t, filepath.Join(dir, "index.json"), index)

	// create channel directory with messages.csv
	chanDir := filepath.Join(dir, "c001")
	os.MkdirAll(chanDir, 0o755)

	csv := "ID,Timestamp,Contents,Attachments\n" +
		"msg1,2023-11-14T15:00:00+00:00,Hello world,\n" +
		"msg2,2023-11-14T16:00:00+00:00,Goodbye world,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", result[0].Text)
	}
	if result[0].Metadata["channel"] != "general" {
		t.Errorf("expected channel 'general', got %v", result[0].Metadata["channel"])
	}
}

func TestImportDiscord_ChannelNameResolution(t *testing.T) {
	dir := t.TempDir()

	// index with channel name
	index := map[string]*string{}
	name := "my-channel"
	index["ch1"] = &name
	writeJSON(t, filepath.Join(dir, "index.json"), index)

	chanDir := filepath.Join(dir, "ch1")
	os.MkdirAll(chanDir, 0o755)

	csv := "ID,Timestamp,Contents,Attachments\nmsg1,2023-11-14T15:00:00+00:00,test,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Metadata["channel"] != "my-channel" {
		t.Errorf("expected channel 'my-channel', got %v", result[0].Metadata["channel"])
	}
}

func TestImportDiscord_FallbackChannelID(t *testing.T) {
	dir := t.TempDir()

	// no index.json, channel name should fallback to directory name
	chanDir := filepath.Join(dir, "ch99")
	os.MkdirAll(chanDir, 0o755)

	csv := "ID,Timestamp,Contents,Attachments\nmsg1,2023-11-14T15:00:00+00:00,test,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Metadata["channel"] != "ch99" {
		t.Errorf("expected channel 'ch99', got %v", result[0].Metadata["channel"])
	}
}

func TestImportDiscord_NullChannelName(t *testing.T) {
	dir := t.TempDir()

	// index with null channel name
	indexJSON := `{"ch1": null}`
	os.WriteFile(filepath.Join(dir, "index.json"), []byte(indexJSON), 0o644)

	chanDir := filepath.Join(dir, "ch1")
	os.MkdirAll(chanDir, 0o755)

	csv := "ID,Timestamp,Contents,Attachments\nmsg1,2023-11-14T15:00:00+00:00,test,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	// null name means fallback to channel ID
	if result[0].Metadata["channel"] != "ch1" {
		t.Errorf("expected channel 'ch1' (fallback), got %v", result[0].Metadata["channel"])
	}
}

func TestImportDiscord_CSVQuotedFields(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "ch1")
	os.MkdirAll(chanDir, 0o755)

	// CSV with quoted fields containing commas
	csv := "ID,Timestamp,Contents,Attachments\n" +
		"msg1,2023-11-14T15:00:00+00:00,\"Hello, world\",\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "Hello, world" {
		t.Errorf("expected 'Hello, world', got %q", result[0].Text)
	}
}

func TestImportDiscord_SinceFilter(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "ch1")
	os.MkdirAll(chanDir, 0o755)

	csv := "ID,Timestamp,Contents,Attachments\n" +
		"old,2020-01-01T00:00:00+00:00,Old message,\n" +
		"new,2023-11-14T15:00:00+00:00,New message,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ImportDiscord(DiscordImportOptions{File: dir, Since: &since})
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

func TestImportDiscord_EmptyContent(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "ch1")
	os.MkdirAll(chanDir, 0o755)

	csv := "ID,Timestamp,Contents,Attachments\n" +
		"msg1,2023-11-14T15:00:00+00:00,,\n" +
		"msg2,2023-11-14T16:00:00+00:00,Has content,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message (empty skipped), got %d", len(result))
	}
}

func TestImportDiscord_MessagesSubdir(t *testing.T) {
	dir := t.TempDir()

	// put everything inside messages/ subdirectory
	msgDir := filepath.Join(dir, "messages")
	os.MkdirAll(msgDir, 0o755)

	index := map[string]*string{}
	ch := "test-channel"
	index["ch1"] = &ch
	data, _ := json.Marshal(index)
	os.WriteFile(filepath.Join(msgDir, "index.json"), data, 0o644)

	chanDir := filepath.Join(msgDir, "ch1")
	os.MkdirAll(chanDir, 0o755)

	csv := "ID,Timestamp,Contents,Attachments\nmsg1,2023-11-14T15:00:00+00:00,Nested,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Metadata["channel"] != "test-channel" {
		t.Errorf("expected 'test-channel', got %v", result[0].Metadata["channel"])
	}
}

func TestImportDiscord_AlternateTimestampFormat(t *testing.T) {
	dir := t.TempDir()

	chanDir := filepath.Join(dir, "ch1")
	os.MkdirAll(chanDir, 0o755)

	// use "2006-01-02 15:04:05" format
	csv := "ID,Timestamp,Contents,Attachments\nmsg1,2023-11-14 15:00:00,Alt format,\n"
	os.WriteFile(filepath.Join(chanDir, "messages.csv"), []byte(csv), 0o644)

	result, err := ImportDiscord(DiscordImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestImportDiscord_NonexistentDir(t *testing.T) {
	_, err := ImportDiscord(DiscordImportOptions{File: "/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
