package connectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportInstagram_DMParsing(t *testing.T) {
	dir := t.TempDir()

	// create inbox structure: messages/inbox/<conv>/message_1.json
	convDir := filepath.Join(dir, "messages", "inbox", "friend_123")
	os.MkdirAll(convDir, 0o755)

	conv := igConversation{
		Messages: []struct {
			SenderName  string `json:"sender_name"`
			TimestampMs int64  `json:"timestamp_ms"`
			Content     string `json:"content,omitempty"`
			Type        string `json:"type,omitempty"`
		}{
			{SenderName: "myuser", TimestampMs: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).UnixMilli(), Content: "Hello from me"},
			{SenderName: "friend", TimestampMs: time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC).UnixMilli(), Content: "Hello from friend"},
		},
	}
	conv.Participants = []struct {
		Name string `json:"name"`
	}{{Name: "myuser"}, {Name: "friend"}}
	writeJSON(t, filepath.Join(convDir, "message_1.json"), conv)

	result, err := ImportInstagram(InstagramImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 DMs, got %d", len(result))
	}
}

func TestImportInstagram_DMUsernameFilter(t *testing.T) {
	dir := t.TempDir()

	convDir := filepath.Join(dir, "messages", "inbox", "friend_123")
	os.MkdirAll(convDir, 0o755)

	conv := igConversation{
		Messages: []struct {
			SenderName  string `json:"sender_name"`
			TimestampMs int64  `json:"timestamp_ms"`
			Content     string `json:"content,omitempty"`
			Type        string `json:"type,omitempty"`
		}{
			{SenderName: "myuser", TimestampMs: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).UnixMilli(), Content: "Mine"},
			{SenderName: "friend", TimestampMs: time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC).UnixMilli(), Content: "Theirs"},
		},
	}
	writeJSON(t, filepath.Join(convDir, "message_1.json"), conv)

	result, err := ImportInstagram(InstagramImportOptions{File: dir, Username: "myuser"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 DM, got %d", len(result))
	}
	if result[0].Text != "Mine" {
		t.Errorf("expected 'Mine', got %q", result[0].Text)
	}
}

func TestImportInstagram_PostCaptionExtraction(t *testing.T) {
	dir := t.TempDir()

	contentDir := filepath.Join(dir, "content")
	os.MkdirAll(contentDir, 0o755)

	posts := []igPost{
		{Title: "Beautiful sunset", CreationTimestamp: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).Unix()},
		{Title: "", Media: []struct {
			Title             string `json:"title,omitempty"`
			CreationTimestamp int64  `json:"creation_timestamp,omitempty"`
		}{{Title: "From media", CreationTimestamp: time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC).Unix()}}},
	}
	writeJSON(t, filepath.Join(contentDir, "posts_1.json"), posts)

	result, err := ImportInstagram(InstagramImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(result))
	}
	if result[0].Text != "Beautiful sunset" {
		t.Errorf("expected 'Beautiful sunset', got %q", result[0].Text)
	}
	if result[1].Text != "From media" {
		t.Errorf("expected 'From media', got %q", result[1].Text)
	}
}

func TestImportInstagram_PostNoCaptionSkipped(t *testing.T) {
	dir := t.TempDir()

	contentDir := filepath.Join(dir, "content")
	os.MkdirAll(contentDir, 0o755)

	posts := []igPost{
		{Title: "", CreationTimestamp: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).Unix()},
	}
	writeJSON(t, filepath.Join(contentDir, "posts_1.json"), posts)

	result, err := ImportInstagram(InstagramImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 posts (no caption), got %d", len(result))
	}
}

func TestImportInstagram_CommentExtraction(t *testing.T) {
	dir := t.TempDir()

	commentsDir := filepath.Join(dir, "comments")
	os.MkdirAll(commentsDir, 0o755)

	comments := []igComment{
		{
			StringMapData: map[string]struct {
				Value     string `json:"value,omitempty"`
				Timestamp int64  `json:"timestamp,omitempty"`
			}{
				"Comment":     {Value: "Nice photo!", Timestamp: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).Unix()},
				"Media Owner": {Value: "someone"},
			},
		},
	}
	writeJSON(t, filepath.Join(commentsDir, "post_comments_1.json"), comments)

	result, err := ImportInstagram(InstagramImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
	if result[0].Text != "Nice photo!" {
		t.Errorf("expected 'Nice photo!', got %q", result[0].Text)
	}
	if result[0].Metadata["mediaOwner"] != "someone" {
		t.Errorf("expected mediaOwner 'someone', got %v", result[0].Metadata["mediaOwner"])
	}
}

func TestImportInstagram_CommentsWrapperFormat(t *testing.T) {
	dir := t.TempDir()

	commentsDir := filepath.Join(dir, "comments")
	os.MkdirAll(commentsDir, 0o755)

	wrapper := igCommentsWrapper{
		CommentsMediaComments: []igComment{
			{
				StringMapData: map[string]struct {
					Value     string `json:"value,omitempty"`
					Timestamp int64  `json:"timestamp,omitempty"`
				}{
					"Comment": {Value: "Wrapped comment", Timestamp: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).Unix()},
				},
			},
		},
	}
	writeJSON(t, filepath.Join(commentsDir, "post_comments_1.json"), wrapper)

	result, err := ImportInstagram(InstagramImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
	if result[0].Text != "Wrapped comment" {
		t.Errorf("expected 'Wrapped comment', got %q", result[0].Text)
	}
}

func TestImportInstagram_SinceFilter(t *testing.T) {
	dir := t.TempDir()

	convDir := filepath.Join(dir, "messages", "inbox", "friend_123")
	os.MkdirAll(convDir, 0o755)

	conv := igConversation{
		Messages: []struct {
			SenderName  string `json:"sender_name"`
			TimestampMs int64  `json:"timestamp_ms"`
			Content     string `json:"content,omitempty"`
			Type        string `json:"type,omitempty"`
		}{
			{SenderName: "me", TimestampMs: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(), Content: "Old"},
			{SenderName: "me", TimestampMs: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).UnixMilli(), Content: "New"},
		},
	}
	writeJSON(t, filepath.Join(convDir, "message_1.json"), conv)

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ImportInstagram(InstagramImportOptions{File: dir, Since: &since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Text != "New" {
		t.Errorf("expected 'New', got %q", result[0].Text)
	}
}

func TestImportInstagram_EmptyContentSkipped(t *testing.T) {
	dir := t.TempDir()

	convDir := filepath.Join(dir, "messages", "inbox", "friend_123")
	os.MkdirAll(convDir, 0o755)

	conv := igConversation{
		Messages: []struct {
			SenderName  string `json:"sender_name"`
			TimestampMs int64  `json:"timestamp_ms"`
			Content     string `json:"content,omitempty"`
			Type        string `json:"type,omitempty"`
		}{
			{SenderName: "me", TimestampMs: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).UnixMilli(), Content: ""},
			{SenderName: "me", TimestampMs: time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC).UnixMilli(), Content: "Real message"},
		},
	}
	writeJSON(t, filepath.Join(convDir, "message_1.json"), conv)

	result, err := ImportInstagram(InstagramImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestImportInstagram_DecodeIGText(t *testing.T) {
	// Test basic ASCII passthrough
	if got := decodeIGText("hello"); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}

	// Test ASCII-range escapes (single-byte values work straightforwardly)
	input := "hello\\u0021"
	got := decodeIGText(input)
	if got != "hello!" {
		t.Errorf("decodeIGText(%q) = %q, want 'hello!'", input, got)
	}

	// Test that non-escaped text passes through unchanged
	if got := decodeIGText("plain text 123"); got != "plain text 123" {
		t.Errorf("expected 'plain text 123', got %q", got)
	}
}

func TestImportInstagram_AllDataTypes(t *testing.T) {
	dir := t.TempDir()

	// DMs
	convDir := filepath.Join(dir, "messages", "inbox", "conv1")
	os.MkdirAll(convDir, 0o755)
	conv := igConversation{
		Messages: []struct {
			SenderName  string `json:"sender_name"`
			TimestampMs int64  `json:"timestamp_ms"`
			Content     string `json:"content,omitempty"`
			Type        string `json:"type,omitempty"`
		}{
			{SenderName: "me", TimestampMs: time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).UnixMilli(), Content: "DM text"},
		},
	}
	writeJSON(t, filepath.Join(convDir, "message_1.json"), conv)

	// Posts
	contentDir := filepath.Join(dir, "content")
	os.MkdirAll(contentDir, 0o755)
	posts := []igPost{
		{Title: "Post caption", CreationTimestamp: time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC).Unix()},
	}
	writeJSON(t, filepath.Join(contentDir, "posts_1.json"), posts)

	// Comments
	commentsDir := filepath.Join(dir, "comments")
	os.MkdirAll(commentsDir, 0o755)
	comments := []igComment{
		{
			StringMapData: map[string]struct {
				Value     string `json:"value,omitempty"`
				Timestamp int64  `json:"timestamp,omitempty"`
			}{
				"Comment": {Value: "A comment", Timestamp: time.Date(2023, 11, 14, 17, 0, 0, 0, time.UTC).Unix()},
			},
		},
	}
	data, _ := json.Marshal(comments)
	os.WriteFile(filepath.Join(commentsDir, "post_comments_1.json"), data, 0o644)

	result, err := ImportInstagram(InstagramImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 messages total (dm+post+comment), got %d", len(result))
	}

	types := map[string]bool{}
	for _, m := range result {
		types[m.Metadata["type"].(string)] = true
	}
	for _, expected := range []string{"dm", "post", "comment"} {
		if !types[expected] {
			t.Errorf("expected type %q in results", expected)
		}
	}
}

func TestImportInstagram_NonexistentDir(t *testing.T) {
	_, err := ImportInstagram(InstagramImportOptions{File: "/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
