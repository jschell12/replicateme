package connectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportTikTok_CommentParsing(t *testing.T) {
	dir := t.TempDir()

	commentsDir := filepath.Join(dir, "Activity", "Comments")
	os.MkdirAll(commentsDir, 0o755)

	data := tikTokCommentList{
		CommentList: []struct {
			Date    string `json:"Date"`
			Comment string `json:"Comment"`
		}{
			{Date: "2023-11-14 15:00:00", Comment: "Great video!"},
			{Date: "2023-11-14 16:00:00", Comment: "Love this"},
		},
	}
	writeJSON(t, filepath.Join(commentsDir, "comments.json"), data)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(result))
	}
	if result[0].Text != "Great video!" {
		t.Errorf("expected 'Great video!', got %q", result[0].Text)
	}
	if result[0].Metadata["type"] != "comment" {
		t.Errorf("expected type 'comment', got %v", result[0].Metadata["type"])
	}
}

func TestImportTikTok_AlternateCommentPaths(t *testing.T) {
	// Test the "Comment/Comments.json" path
	dir := t.TempDir()

	commentsDir := filepath.Join(dir, "Comment")
	os.MkdirAll(commentsDir, 0o755)

	data := tikTokCommentList{
		CommentList: []struct {
			Date    string `json:"Date"`
			Comment string `json:"Comment"`
		}{
			{Date: "2023-11-14 15:00:00", Comment: "Alt path comment"},
		},
	}
	writeJSON(t, filepath.Join(commentsDir, "Comments.json"), data)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
	if result[0].Text != "Alt path comment" {
		t.Errorf("expected 'Alt path comment', got %q", result[0].Text)
	}
}

func TestImportTikTok_RootCommentPath(t *testing.T) {
	dir := t.TempDir()

	data := tikTokCommentList{
		CommentList: []struct {
			Date    string `json:"Date"`
			Comment string `json:"Comment"`
		}{
			{Date: "2023-11-14 15:00:00", Comment: "Root comment"},
		},
	}
	writeJSON(t, filepath.Join(dir, "comments.json"), data)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
}

func TestImportTikTok_DMParsing(t *testing.T) {
	dir := t.TempDir()

	dmDir := filepath.Join(dir, "Activity", "Direct Messages")
	convDir := filepath.Join(dmDir, "conv1")
	os.MkdirAll(convDir, 0o755)

	conv := tikTokDMConversation{
		Messages: []struct {
			Date    string `json:"Date"`
			From    string `json:"From"`
			Content string `json:"Content"`
		}{
			{Date: "2023-11-14 15:00:00", From: "myuser", Content: "Hey!"},
			{Date: "2023-11-14 16:00:00", From: "friend", Content: "Hi there"},
		},
	}
	writeJSON(t, filepath.Join(convDir, "chat.json"), conv)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 DMs, got %d", len(result))
	}
	if result[0].Text != "Hey!" {
		t.Errorf("expected 'Hey!', got %q", result[0].Text)
	}
}

func TestImportTikTok_DMUsernameFilter(t *testing.T) {
	dir := t.TempDir()

	dmDir := filepath.Join(dir, "Activity", "Direct Messages")
	convDir := filepath.Join(dmDir, "conv1")
	os.MkdirAll(convDir, 0o755)

	conv := tikTokDMConversation{
		Messages: []struct {
			Date    string `json:"Date"`
			From    string `json:"From"`
			Content string `json:"Content"`
		}{
			{Date: "2023-11-14 15:00:00", From: "myuser", Content: "Mine"},
			{Date: "2023-11-14 16:00:00", From: "other", Content: "Theirs"},
		},
	}
	writeJSON(t, filepath.Join(convDir, "chat.json"), conv)

	result, err := ImportTikTok(TikTokImportOptions{File: dir, Username: "myuser"})
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

func TestImportTikTok_DMAsDirectFile(t *testing.T) {
	dir := t.TempDir()

	// DM JSON files directly in the DM directory (not in subdirectory)
	dmDir := filepath.Join(dir, "Direct Messages")
	os.MkdirAll(dmDir, 0o755)

	conv := tikTokDMConversation{
		Messages: []struct {
			Date    string `json:"Date"`
			From    string `json:"From"`
			Content string `json:"Content"`
		}{
			{Date: "2023-11-14 15:00:00", From: "user1", Content: "Direct file DM"},
		},
	}
	writeJSON(t, filepath.Join(dmDir, "conversation.json"), conv)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 DM, got %d", len(result))
	}
}

func TestImportTikTok_SinceFilter(t *testing.T) {
	dir := t.TempDir()

	commentsDir := filepath.Join(dir, "Activity", "Comments")
	os.MkdirAll(commentsDir, 0o755)

	data := tikTokCommentList{
		CommentList: []struct {
			Date    string `json:"Date"`
			Comment string `json:"Comment"`
		}{
			{Date: "2020-01-01 00:00:00", Comment: "Old"},
			{Date: "2023-11-14 15:00:00", Comment: "New"},
		},
	}
	writeJSON(t, filepath.Join(commentsDir, "comments.json"), data)

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ImportTikTok(TikTokImportOptions{File: dir, Since: &since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
	if result[0].Text != "New" {
		t.Errorf("expected 'New', got %q", result[0].Text)
	}
}

func TestImportTikTok_EmptyCommentSkipped(t *testing.T) {
	dir := t.TempDir()

	commentsDir := filepath.Join(dir, "Activity", "Comments")
	os.MkdirAll(commentsDir, 0o755)

	data := tikTokCommentList{
		CommentList: []struct {
			Date    string `json:"Date"`
			Comment string `json:"Comment"`
		}{
			{Date: "2023-11-14 15:00:00", Comment: ""},
			{Date: "2023-11-14 16:00:00", Comment: "Real comment"},
		},
	}
	writeJSON(t, filepath.Join(commentsDir, "comments.json"), data)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
}

func TestImportTikTok_EmptyDMContentSkipped(t *testing.T) {
	dir := t.TempDir()

	dmDir := filepath.Join(dir, "Direct Messages")
	os.MkdirAll(dmDir, 0o755)

	conv := tikTokDMConversation{
		Messages: []struct {
			Date    string `json:"Date"`
			From    string `json:"From"`
			Content string `json:"Content"`
		}{
			{Date: "2023-11-14 15:00:00", From: "user1", Content: ""},
			{Date: "2023-11-14 16:00:00", From: "user1", Content: "Real DM"},
		},
	}
	writeJSON(t, filepath.Join(dmDir, "conv.json"), conv)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 DM, got %d", len(result))
	}
}

func TestParseTikTokDate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"2023-11-14 15:00:00", true},
		{"2023-11-14T15:00:00Z", true},
		{"2023-11-14T15:00:00", true},
		{"2023-11-14", true},
		{"not-a-date", false},
		{"", false},
	}

	for _, tc := range tests {
		_, err := parseTikTokDate(tc.input)
		if tc.valid && err != nil {
			t.Errorf("parseTikTokDate(%q) unexpected error: %v", tc.input, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("parseTikTokDate(%q) expected error, got nil", tc.input)
		}
	}
}

func TestImportTikTok_CommentsAndDMs(t *testing.T) {
	dir := t.TempDir()

	// comments
	commentsDir := filepath.Join(dir, "Activity", "Comments")
	os.MkdirAll(commentsDir, 0o755)
	cl := tikTokCommentList{
		CommentList: []struct {
			Date    string `json:"Date"`
			Comment string `json:"Comment"`
		}{
			{Date: "2023-11-14 15:00:00", Comment: "A comment"},
		},
	}
	cData, _ := json.Marshal(cl)
	os.WriteFile(filepath.Join(commentsDir, "comments.json"), cData, 0o644)

	// DMs
	dmDir := filepath.Join(dir, "Activity", "Direct Messages")
	convDir := filepath.Join(dmDir, "conv1")
	os.MkdirAll(convDir, 0o755)
	conv := tikTokDMConversation{
		Messages: []struct {
			Date    string `json:"Date"`
			From    string `json:"From"`
			Content string `json:"Content"`
		}{
			{Date: "2023-11-14 16:00:00", From: "me", Content: "A DM"},
		},
	}
	dData, _ := json.Marshal(conv)
	os.WriteFile(filepath.Join(convDir, "chat.json"), dData, 0o644)

	result, err := ImportTikTok(TikTokImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
}

func TestImportTikTok_NonexistentDir(t *testing.T) {
	_, err := ImportTikTok(TikTokImportOptions{File: "/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
