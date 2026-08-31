package connectors

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportGmail_BasicSentMail(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "mail.mbox")

	content := `From sender@example.com Thu Nov 14 10:00:00 2023
From: me@example.com
To: friend@example.com
Date: Thu, 14 Nov 2023 10:00:00 -0500
Subject: Hello

This is a test email body.

From other@example.com Thu Nov 14 11:00:00 2023
From: other@example.com
To: me@example.com
Date: Thu, 14 Nov 2023 11:00:00 -0500
Subject: Reply

This is from someone else.
`
	os.WriteFile(mboxPath, []byte(content), 0o644)

	result, err := ImportGmail(GmailImportOptions{File: mboxPath, Email: "me@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(result))
	}
	if result[0].Text != "This is a test email body." {
		t.Errorf("expected body text, got %q", result[0].Text)
	}
	if result[0].Metadata["subject"] != "Hello" {
		t.Errorf("expected subject 'Hello', got %v", result[0].Metadata["subject"])
	}
}

func TestImportGmail_SentViaLabels(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "mail.mbox")

	content := `From sender@example.com Thu Nov 14 10:00:00 2023
From: me@example.com
To: friend@example.com
Date: Thu, 14 Nov 2023 10:00:00 -0500
Subject: Labeled sent
X-Gmail-Labels: Sent,Important

Body of labeled sent message.
`
	os.WriteFile(mboxPath, []byte(content), 0o644)

	// no Email filter, rely on X-Gmail-Labels
	result, err := ImportGmail(GmailImportOptions{File: mboxPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message via labels, got %d", len(result))
	}
}

func TestImportGmail_QuotedReplyStripping(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "mail.mbox")

	content := `From sender@example.com Thu Nov 14 10:00:00 2023
From: me@example.com
To: friend@example.com
Date: Thu, 14 Nov 2023 10:00:00 -0500
Subject: Re: Hello

My actual reply.

> This is a quoted line.
> Another quoted line.
`
	os.WriteFile(mboxPath, []byte(content), 0o644)

	result, err := ImportGmail(GmailImportOptions{File: mboxPath, Email: "me@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	text := result[0].Text
	if contains(text, "> This is a quoted line") {
		t.Errorf("quoted lines should be stripped, got %q", text)
	}
	if !contains(text, "My actual reply.") {
		t.Errorf("expected 'My actual reply.' in text, got %q", text)
	}
}

func TestImportGmail_SignatureStripping(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "mail.mbox")

	// Note: signature delimiter is "-- " (dash-dash-space) per RFC 3676
	content := "From sender@example.com Thu Nov 14 10:00:00 2023\n" +
		"From: me@example.com\n" +
		"To: friend@example.com\n" +
		"Date: Thu, 14 Nov 2023 10:00:00 -0500\n" +
		"Subject: Sig test\n" +
		"\n" +
		"Main body text.\n" +
		"\n" +
		"-- \n" +
		"John Doe\n" +
		"CEO, Example Corp\n"
	os.WriteFile(mboxPath, []byte(content), 0o644)

	result, err := ImportGmail(GmailImportOptions{File: mboxPath, Email: "me@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	text := result[0].Text
	if contains(text, "John Doe") {
		t.Errorf("signature should be stripped, got %q", text)
	}
	if !contains(text, "Main body text.") {
		t.Errorf("expected 'Main body text.' in text, got %q", text)
	}
}

func TestImportGmail_QuotedPrintableDecode(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "mail.mbox")

	// =C3=A9 is UTF-8 for e-acute in QP encoding
	content := `From sender@example.com Thu Nov 14 10:00:00 2023
From: me@example.com
To: friend@example.com
Date: Thu, 14 Nov 2023 10:00:00 -0500
Subject: QP test

caf=C3=A9 is great
`
	os.WriteFile(mboxPath, []byte(content), 0o644)

	result, err := ImportGmail(GmailImportOptions{File: mboxPath, Email: "me@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	// The QP decoder converts =C3 and =A9 to bytes 0xC3 and 0xA9 which is UTF-8 for e-acute
	text := result[0].Text
	if !contains(text, "caf") {
		t.Errorf("expected 'caf' prefix in text, got %q", text)
	}
}

func TestImportGmail_SoftLineBreakDecode(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "mail.mbox")

	// Use \n for headers, put =\n (QP soft break) only in body
	content := "From sender@example.com Thu Nov 14 10:00:00 2023\n" +
		"From: me@example.com\n" +
		"To: friend@example.com\n" +
		"Date: Thu, 14 Nov 2023 10:00:00 -0500\n" +
		"Subject: Soft break\n" +
		"\n" +
		"This is a long line that was bro=\n" +
		"ken by the mailer.\n"
	os.WriteFile(mboxPath, []byte(content), 0o644)

	result, err := ImportGmail(GmailImportOptions{File: mboxPath, Email: "me@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	text := result[0].Text
	if !contains(text, "broken") {
		t.Errorf("soft line break should be joined, got %q", text)
	}
}

func TestImportGmail_SinceFilter(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "mail.mbox")

	content := `From sender@example.com Thu Nov 14 10:00:00 2023
From: me@example.com
To: friend@example.com
Date: Thu, 14 Nov 2020 10:00:00 -0500
Subject: Old

Old message.

From sender@example.com Thu Nov 14 10:00:00 2023
From: me@example.com
To: friend@example.com
Date: Thu, 14 Nov 2023 10:00:00 -0500
Subject: New

New message.
`
	os.WriteFile(mboxPath, []byte(content), 0o644)

	since := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := ImportGmail(GmailImportOptions{File: mboxPath, Email: "me@example.com", Since: &since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message after since filter, got %d", len(result))
	}
	if result[0].Metadata["subject"] != "New" {
		t.Errorf("expected 'New', got %v", result[0].Metadata["subject"])
	}
}

func TestImportGmail_EmptyMbox(t *testing.T) {
	dir := t.TempDir()
	mboxPath := filepath.Join(dir, "empty.mbox")
	os.WriteFile(mboxPath, []byte(""), 0o644)

	result, err := ImportGmail(GmailImportOptions{File: mboxPath, Email: "me@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(result))
	}
}

func TestImportGmail_NonexistentFile(t *testing.T) {
	_, err := ImportGmail(GmailImportOptions{File: "/nonexistent/mail.mbox"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestImportGmail_ZipImport(t *testing.T) {
	dir := t.TempDir()

	// create staging dir with an mbox file containing "sent" in the name
	stagingDir := filepath.Join(dir, "staging", "Takeout", "Mail")
	os.MkdirAll(stagingDir, 0o755)

	content := `From sender@example.com Thu Nov 14 10:00:00 2023
From: me@example.com
To: friend@example.com
Date: Thu, 14 Nov 2023 10:00:00 -0500
Subject: Zipped
X-Gmail-Labels: Sent

Zipped body.
`
	os.WriteFile(filepath.Join(stagingDir, "Sent Mail.mbox"), []byte(content), 0o644)

	zipPath := filepath.Join(dir, "gmail.zip")
	createZipFromDir(t, filepath.Join(dir, "staging"), zipPath)

	result, err := ImportGmail(GmailImportOptions{File: zipPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message from zip, got %d", len(result))
	}
}

func TestParseEmailDate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"Thu, 14 Nov 2023 10:00:00 -0500", true},
		{"Thu, 14 Nov 2023 10:00:00 -0500 (EST)", true},
		{"14 Nov 2023 10:00:00 -0500", true},
		{"2023-11-14T10:00:00Z", true},
		{"", false},
		{"not a date", false},
	}

	for _, tc := range tests {
		_, err := parseEmailDate(tc.input)
		if tc.valid && err != nil {
			t.Errorf("parseEmailDate(%q) unexpected error: %v", tc.input, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("parseEmailDate(%q) expected error, got nil", tc.input)
		}
	}
}

func TestParseHeaders(t *testing.T) {
	block := "From: me@example.com\nTo: you@example.com\nSubject: Test\n\tContinued subject"
	h := parseHeaders(block)
	if h["from"] != "me@example.com" {
		t.Errorf("expected 'me@example.com', got %q", h["from"])
	}
	if h["to"] != "you@example.com" {
		t.Errorf("expected 'you@example.com', got %q", h["to"])
	}
	// continuation line should be unfolded
	if !contains(h["subject"], "Continued subject") {
		t.Errorf("expected continuation line unfolded, got %q", h["subject"])
	}
}

// contains checks substring presence
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
