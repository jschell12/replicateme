package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportTwitter_BasicTweets(t *testing.T) {
	dir := t.TempDir()

	tweetsJS := `window.YTD.tweet.part0 = [
		{
			"tweet": {
				"id_str": "123",
				"full_text": "Hello world!",
				"created_at": "Tue Nov 14 15:00:00 +0000 2023"
			}
		},
		{
			"tweet": {
				"id_str": "456",
				"full_text": "Second tweet",
				"created_at": "Tue Nov 14 16:00:00 +0000 2023"
			}
		}
	]`
	os.WriteFile(filepath.Join(dir, "tweets.js"), []byte(tweetsJS), 0o644)

	result, err := ImportTwitter(TwitterImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 tweets, got %d", len(result))
	}
	if result[0].Text != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", result[0].Text)
	}
	if result[0].ID != "twitter-123" {
		t.Errorf("expected ID 'twitter-123', got %q", result[0].ID)
	}
}

func TestImportTwitter_DMParsing(t *testing.T) {
	dir := t.TempDir()

	// empty tweets file
	os.WriteFile(filepath.Join(dir, "tweets.js"), []byte("window.YTD.tweet.part0 = []"), 0o644)

	// DMs with millisecond timestamps
	dmsJS := fmt.Sprintf(`window.YTD.direct_messages.part0 = [
		{
			"dmConversation": {
				"messages": [
					{
						"messageCreate": {
							"id": "dm1",
							"text": "Hey there",
							"createdAt": "%d",
							"senderId": "111",
							"recipientId": "222"
						}
					},
					{
						"messageCreate": {
							"id": "dm2",
							"text": "How are you?",
							"createdAt": "%d",
							"senderId": "222",
							"recipientId": "111"
						}
					}
				]
			}
		}
	]`, time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).UnixMilli(),
		time.Date(2023, 11, 14, 16, 0, 0, 0, time.UTC).UnixMilli())
	os.WriteFile(filepath.Join(dir, "direct-messages.js"), []byte(dmsJS), 0o644)

	result, err := ImportTwitter(TwitterImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 DMs, got %d", len(result))
	}
	if result[0].Text != "Hey there" {
		t.Errorf("expected 'Hey there', got %q", result[0].Text)
	}
	if result[0].Metadata["isDM"] != true {
		t.Errorf("expected isDM=true, got %v", result[0].Metadata["isDM"])
	}
}

func TestImportTwitter_TcoURLRemoval(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Check this https://t.co/abc123", "Check this"},
		{"No URL here", "No URL here"},
		{"Multiple https://t.co/a https://t.co/b", "Multiple"},
	}

	for _, tc := range tests {
		got := cleanTweetText(tc.input)
		if got != tc.expected {
			t.Errorf("cleanTweetText(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestImportTwitter_HTMLEntityDecode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Tom &amp; Jerry", "Tom & Jerry"},
		{"5 &lt; 10 &gt; 3", "5 < 10 > 3"},
		{"She said &quot;hello&quot;", `She said "hello"`},
	}

	for _, tc := range tests {
		got := cleanTweetText(tc.input)
		if got != tc.expected {
			t.Errorf("cleanTweetText(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestImportTwitter_SinceFilter(t *testing.T) {
	dir := t.TempDir()

	tweetsJS := `window.YTD.tweet.part0 = [
		{
			"tweet": {
				"id_str": "old",
				"full_text": "Old tweet",
				"created_at": "Mon Jan 01 00:00:00 +0000 2020"
			}
		},
		{
			"tweet": {
				"id_str": "new",
				"full_text": "New tweet",
				"created_at": "Tue Nov 14 15:00:00 +0000 2023"
			}
		}
	]`
	os.WriteFile(filepath.Join(dir, "tweets.js"), []byte(tweetsJS), 0o644)

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ImportTwitter(TwitterImportOptions{File: dir, Since: &since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(result))
	}
	if result[0].ID != "twitter-new" {
		t.Errorf("expected 'twitter-new', got %q", result[0].ID)
	}
}

func TestImportTwitter_ReplyDetection(t *testing.T) {
	dir := t.TempDir()

	tweetsJS := `window.YTD.tweet.part0 = [
		{
			"tweet": {
				"id_str": "reply1",
				"full_text": "@someone replying here",
				"created_at": "Tue Nov 14 15:00:00 +0000 2023",
				"in_reply_to_status_id_str": "orig1",
				"in_reply_to_screen_name": "someone"
			}
		},
		{
			"tweet": {
				"id_str": "orig1",
				"full_text": "Original tweet",
				"created_at": "Tue Nov 14 14:00:00 +0000 2023"
			}
		}
	]`
	os.WriteFile(filepath.Join(dir, "tweets.js"), []byte(tweetsJS), 0o644)

	result, err := ImportTwitter(TwitterImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 tweets, got %d", len(result))
	}

	// find the reply
	var reply *struct {
		isReply bool
		replyTo string
	}
	for _, msg := range result {
		if msg.ID == "twitter-reply1" {
			ir, _ := msg.Metadata["isReply"].(bool)
			rt, _ := msg.Metadata["replyTo"].(string)
			reply = &struct {
				isReply bool
				replyTo string
			}{ir, rt}
		}
	}
	if reply == nil {
		t.Fatal("reply tweet not found")
	}
	if !reply.isReply {
		t.Error("expected isReply=true")
	}
	if reply.replyTo != "someone" {
		t.Errorf("expected replyTo='someone', got %q", reply.replyTo)
	}
}

func TestImportTwitter_IncludeDMsFalse(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "tweets.js"), []byte("window.YTD.tweet.part0 = []"), 0o644)

	dmsJS := fmt.Sprintf(`window.YTD.direct_messages.part0 = [
		{
			"dmConversation": {
				"messages": [
					{
						"messageCreate": {
							"id": "dm1",
							"text": "A DM",
							"createdAt": "%d",
							"senderId": "111",
							"recipientId": "222"
						}
					}
				]
			}
		}
	]`, time.Date(2023, 11, 14, 15, 0, 0, 0, time.UTC).UnixMilli())
	os.WriteFile(filepath.Join(dir, "direct-messages.js"), []byte(dmsJS), 0o644)

	includeDMs := false
	result, err := ImportTwitter(TwitterImportOptions{File: dir, IncludeDMs: &includeDMs})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 messages with DMs disabled, got %d", len(result))
	}
}

func TestImportTwitter_DataSubdir(t *testing.T) {
	dir := t.TempDir()

	// put tweets.js inside data/ subdirectory
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)

	tweetsJS := `window.YTD.tweet.part0 = [
		{
			"tweet": {
				"id_str": "1",
				"full_text": "From data dir",
				"created_at": "Tue Nov 14 15:00:00 +0000 2023"
			}
		}
	]`
	os.WriteFile(filepath.Join(dataDir, "tweets.js"), []byte(tweetsJS), 0o644)

	result, err := ImportTwitter(TwitterImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(result))
	}
}

func TestImportTwitter_EmptyFullText(t *testing.T) {
	dir := t.TempDir()

	tweetsJS := `window.YTD.tweet.part0 = [
		{
			"tweet": {
				"id_str": "1",
				"full_text": "",
				"created_at": "Tue Nov 14 15:00:00 +0000 2023"
			}
		}
	]`
	os.WriteFile(filepath.Join(dir, "tweets.js"), []byte(tweetsJS), 0o644)

	result, err := ImportTwitter(TwitterImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 tweets with empty text, got %d", len(result))
	}
}

func TestImportTwitter_NonexistentDir(t *testing.T) {
	_, err := ImportTwitter(TwitterImportOptions{File: "/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
