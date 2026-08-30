package corpus

import "time"

// Platform represents a message source platform.
type Platform string

const (
	IMMessage Platform = "imessage"
	Slack     Platform = "slack"
	Email     Platform = "email"
	GitHub    Platform = "github"
	Twitter   Platform = "twitter"
	Discord   Platform = "discord"
	Reddit    Platform = "reddit"
	Instagram Platform = "instagram"
	TikTok    Platform = "tiktok"
)

// RawMessage is a single message ingested from any platform.
type RawMessage struct {
	ID         string            `json:"id"`
	Text       string            `json:"text"`
	Platform   Platform          `json:"platform"`
	Timestamp  time.Time         `json:"timestamp"`
	IsFromUser bool              `json:"isFromUser"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

// StyleProfile captures quantitative writing-style characteristics.
type StyleProfile struct {
	AverageLength         int            `json:"averageLength"`
	CapitalizesFirstWord  float64        `json:"capitalizesFirstWord"`
	UsesContractions      float64        `json:"usesContractions"`
	UsesPeriods           float64        `json:"usesPeriods"`
	UsesCommas            float64        `json:"usesCommas"`
	UsesExclamation       float64        `json:"usesExclamation"`
	UsesQuestionMark      float64        `json:"usesQuestionMark"`
	UsesEmoji             float64        `json:"usesEmoji"`
	CommonPhrases         []string       `json:"commonPhrases"`
	TypicalErrors         []TypicalError `json:"typicalErrors"`
	SentenceFragmentRatio float64        `json:"sentenceFragmentRatio"`
	LowercaseIRatio       float64        `json:"lowercaseIRatio"`
}

// TypicalError describes a recurring writing mistake or quirk.
type TypicalError struct {
	Pattern   string   `json:"pattern"`
	Frequency int      `json:"frequency"`
	Examples  []string `json:"examples"`
}

// CorpusStats summarizes what's stored in the corpus database.
type CorpusStats struct {
	TotalMessages int               `json:"totalMessages"`
	ByPlatform    []PlatformCount   `json:"byPlatform"`
	Profiles      []ProfileSummary  `json:"profiles"`
}

// PlatformCount is a platform name paired with a message count.
type PlatformCount struct {
	Platform string `json:"platform"`
	Count    int    `json:"count"`
}

// ProfileSummary is a lightweight view of a stored profile row.
type ProfileSummary struct {
	Platform     string `json:"platform"`
	MessageCount int    `json:"messageCount"`
	UpdatedAt    string `json:"updatedAt"`
}
