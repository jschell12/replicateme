package generate

import (
	"fmt"
	"strings"

	"github.com/jschell12/replicateme/pkg/corpus"
	"github.com/jschell12/replicateme/pkg/style"
)

type GenerateOptions struct {
	Platform        string
	Context         string
	SimilarMessages []corpus.RawMessage
	StyleProfile    corpus.StyleProfile
	QuirkLevel      int
	Instruction     string
}

func BuildSystemPrompt(opts GenerateOptions) string {
	var b strings.Builder

	b.WriteString("You are a writing style replicator. Your job is to write messages that sound exactly like a specific person based on their writing patterns and examples.\n\n")
	b.WriteString("You will be given their style profile (statistical patterns), example messages they have written, and context for what needs to be written. Your output should be indistinguishable from something they would actually write.\n\n")

	fmt.Fprintf(&b, "## Platform: %s\n\n", opts.Platform)
	b.WriteString(platformGuidance(opts.Platform))
	b.WriteString("\n\n")

	b.WriteString(style.StyleProfileToPrompt(opts.StyleProfile))
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "## Quirk level: %d%%\n\n", opts.QuirkLevel)
	switch {
	case opts.QuirkLevel == 0:
		b.WriteString("Use their vocabulary and voice but with clean grammar and spelling. No intentional errors.\n")
	case opts.QuirkLevel <= 30:
		b.WriteString("Mostly clean writing but include their most common habits like missing apostrophes in contractions (im, dont, cant) occasionally.\n")
	case opts.QuirkLevel <= 70:
		b.WriteString("Write naturally in their style including their typical quirks: missing apostrophes, lowercase starts, fragments, and their common phrases. This should read like their normal messages.\n")
	default:
		b.WriteString("Full authenticity. Include all their writing quirks: missing apostrophes, lowercase i, sentence fragments, double spaces, repeated words, minimal punctuation. This should be indistinguishable from their actual messages.\n")
	}

	if len(opts.SimilarMessages) > 0 {
		b.WriteString("\n## Example messages from this person\n\n")
		limit := 20
		if len(opts.SimilarMessages) < limit {
			limit = len(opts.SimilarMessages)
		}
		for i := 0; i < limit; i++ {
			fmt.Fprintf(&b, "- \"%s\"\n", opts.SimilarMessages[i].Text)
		}
	}

	b.WriteString("\n## Rules\n\n")
	b.WriteString("- Output ONLY the message text, nothing else\n")
	b.WriteString("- Do not add quotes around the message\n")
	b.WriteString("- Do not explain or caveat\n")
	b.WriteString("- Match their typical message length for this type of content\n")
	b.WriteString("- If they rarely use periods, dont add periods. If they rarely capitalize, dont capitalize.\n")
	b.WriteString("- Never use words or phrases this person wouldnt use based on their examples\n")

	return b.String()
}

func BuildUserPrompt(opts GenerateOptions) string {
	var b strings.Builder

	if opts.Instruction != "" {
		fmt.Fprintf(&b, "Task: %s\n\n", opts.Instruction)
	}

	fmt.Fprintf(&b, "Context:\n%s\n\n", opts.Context)
	b.WriteString("Write a response in this person's voice and style. Output only the message.")

	return b.String()
}

func platformGuidance(platform string) string {
	switch platform {
	case "imessage":
		return "This is a text message / iMessage. Messages are typically very short, casual, and conversational. Multiple short messages are common instead of one long one."
	case "slack":
		return "This is a Slack message at work. Slightly more structured than texts but still casual. May reference channels, threads, or colleagues."
	case "email":
		return "This is an email. More structured than chat but still in the person's voice. Includes greeting and sign-off only if the person typically uses them."
	case "github":
		return "This is a GitHub comment, PR description, or commit message. Technical and concise. Follows the person's typical commit/PR style."
	case "twitter":
		return "This is a tweet or reply. Very concise (280 char limit). Matches the person's typical Twitter tone."
	case "discord":
		return "This is a Discord message. Similar to texting but may be in a server with specific context."
	case "reddit":
		return "This is a Reddit post or comment. May be longer and more detailed than texts but still in the person's voice."
	default:
		return "Write in the person's natural style for this platform."
	}
}
