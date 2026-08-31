package connectors

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportReddit_BasicComments(t *testing.T) {
	dir := t.TempDir()

	// columns: id,permalink,date,ip,subreddit,gildings,link,parent,body,media
	csv := "id,permalink,date,ip,subreddit,gildings,link,parent,body,media\n" +
		"c1,/r/golang/1,2023-11-14T15:00:00+00:00,,golang,,,,This is my comment,\n" +
		"c2,/r/rust/2,2023-11-14T16:00:00+00:00,,rust,,,,Another comment,\n"
	os.WriteFile(filepath.Join(dir, "comments.csv"), []byte(csv), 0o644)

	result, err := ImportReddit(RedditImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(result))
	}
	if result[0].Text != "This is my comment" {
		t.Errorf("expected 'This is my comment', got %q", result[0].Text)
	}
	if result[0].Metadata["subreddit"] != "golang" {
		t.Errorf("expected subreddit 'golang', got %v", result[0].Metadata["subreddit"])
	}
	if result[0].Metadata["type"] != "comment" {
		t.Errorf("expected type 'comment', got %v", result[0].Metadata["type"])
	}
}

func TestImportReddit_BasicPosts(t *testing.T) {
	dir := t.TempDir()

	// columns: id,permalink,date,ip,subreddit,gildings,title,url,body,media
	csv := "id,permalink,date,ip,subreddit,gildings,title,url,body,media\n" +
		"p1,/r/golang/post1,2023-11-14T15:00:00+00:00,,golang,,My Title,,Post body here,\n"
	os.WriteFile(filepath.Join(dir, "posts.csv"), []byte(csv), 0o644)

	result, err := ImportReddit(RedditImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 post, got %d", len(result))
	}
	// title + body joined with double newline
	if result[0].Text != "My Title\n\nPost body here" {
		t.Errorf("expected 'My Title\\n\\nPost body here', got %q", result[0].Text)
	}
	if result[0].Metadata["type"] != "post" {
		t.Errorf("expected type 'post', got %v", result[0].Metadata["type"])
	}
	if result[0].Metadata["title"] != "My Title" {
		t.Errorf("expected title 'My Title', got %v", result[0].Metadata["title"])
	}
}

func TestImportReddit_PostTitleOnly(t *testing.T) {
	dir := t.TempDir()

	// post with title but no body
	csv := "id,permalink,date,ip,subreddit,gildings,title,url,body,media\n" +
		"p1,/r/golang/post1,2023-11-14T15:00:00+00:00,,golang,,Title Only,,,\n"
	os.WriteFile(filepath.Join(dir, "posts.csv"), []byte(csv), 0o644)

	result, err := ImportReddit(RedditImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 post, got %d", len(result))
	}
	if result[0].Text != "Title Only" {
		t.Errorf("expected 'Title Only', got %q", result[0].Text)
	}
}

func TestImportReddit_HTMLEntityDecode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Tom &amp; Jerry", "Tom & Jerry"},
		{"5 &lt; 10 &gt; 3", "5 < 10 > 3"},
		{"&quot;quoted&quot;", `"quoted"`},
		{"it&#39;s fine", "it's fine"},
	}

	for _, tc := range tests {
		got := cleanRedditText(tc.input)
		if got != tc.expected {
			t.Errorf("cleanRedditText(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestImportReddit_MarkdownLinkStripping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[click here](https://example.com)", "click here"},
		{"Check [this](http://x.com) out", "Check this out"},
		{"No links here", "No links here"},
	}

	for _, tc := range tests {
		got := cleanRedditText(tc.input)
		if got != tc.expected {
			t.Errorf("cleanRedditText(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestImportReddit_SinceFilter(t *testing.T) {
	dir := t.TempDir()

	csv := "id,permalink,date,ip,subreddit,gildings,link,parent,body,media\n" +
		"c1,/r/old/1,2020-01-01T00:00:00+00:00,,old,,,,Old comment,\n" +
		"c2,/r/new/2,2023-11-14T15:00:00+00:00,,new,,,,New comment,\n"
	os.WriteFile(filepath.Join(dir, "comments.csv"), []byte(csv), 0o644)

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ImportReddit(RedditImportOptions{File: dir, Since: &since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
	if result[0].Text != "New comment" {
		t.Errorf("expected 'New comment', got %q", result[0].Text)
	}
}

func TestImportReddit_CommentsAndPostsCombined(t *testing.T) {
	dir := t.TempDir()

	comments := "id,permalink,date,ip,subreddit,gildings,link,parent,body,media\n" +
		"c1,/r/x/1,2023-11-14T16:00:00+00:00,,x,,,,A comment,\n"
	os.WriteFile(filepath.Join(dir, "comments.csv"), []byte(comments), 0o644)

	posts := "id,permalink,date,ip,subreddit,gildings,title,url,body,media\n" +
		"p1,/r/x/2,2023-11-14T15:00:00+00:00,,x,,A Post,,,\n"
	os.WriteFile(filepath.Join(dir, "posts.csv"), []byte(posts), 0o644)

	result, err := ImportReddit(RedditImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	// sorted by timestamp: post (15:00) before comment (16:00)
	if result[0].ID != "reddit-post-p1" {
		t.Errorf("expected post first, got %q", result[0].ID)
	}
}

func TestImportReddit_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// no CSV files at all
	result, err := ImportReddit(RedditImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(result))
	}
}

func TestImportReddit_AlternateTimestampFormat(t *testing.T) {
	dir := t.TempDir()

	csv := "id,permalink,date,ip,subreddit,gildings,link,parent,body,media\n" +
		"c1,/r/test/1,2023-11-14 15:00:00,,test,,,,Alt format comment,\n"
	os.WriteFile(filepath.Join(dir, "comments.csv"), []byte(csv), 0o644)

	result, err := ImportReddit(RedditImportOptions{File: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
}

func TestImportReddit_NonexistentDir(t *testing.T) {
	_, err := ImportReddit(RedditImportOptions{File: "/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
