package portable

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jschell12/replicateme/pkg/corpus"
	"github.com/jschell12/replicateme/pkg/style"
)

// ExportBundle is the portable file format. Contains style profiles
// and optionally a persona spec. No raw messages are included.
type ExportBundle struct {
	Version     int                        `json:"version"`
	ExportedAt  string                     `json:"exportedAt"`
	Profiles    map[string]ProfileEntry    `json:"profiles"`
	PersonaSpec string                     `json:"personaSpec,omitempty"`
}

// ProfileEntry is a single platform's style profile in the bundle.
type ProfileEntry struct {
	Profile      corpus.StyleProfile `json:"profile"`
	MessageCount int                 `json:"messageCount"`
}

// Export creates a portable bundle from the local corpus.
// Includes all per-platform profiles + combined. No raw messages.
func Export(personaPath string) (*ExportBundle, error) {
	profiles, err := corpus.GetAllProfiles()
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found. Run `replicateme ingest` first")
	}

	bundle := &ExportBundle{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profiles:   make(map[string]ProfileEntry),
	}

	for _, p := range profiles {
		bundle.Profiles[p.Platform] = ProfileEntry{
			Profile:      p.Profile,
			MessageCount: p.MessageCount,
		}
	}

	if personaPath != "" {
		data, err := os.ReadFile(personaPath)
		if err != nil {
			return nil, fmt.Errorf("reading persona file: %w", err)
		}
		bundle.PersonaSpec = string(data)
	}

	return bundle, nil
}

// ExportToFile writes the bundle to a JSON file.
func ExportToFile(path string, personaPath string) error {
	bundle, err := Export(personaPath)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0644)
}

// Import loads a bundle and merges its profiles into the local corpus.
// Does not overwrite existing profiles with more data - keeps the one
// with the higher message count.
func Import(path string) (*ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var bundle ExportBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("invalid bundle file: %w", err)
	}

	result := &ImportResult{}

	for platform, entry := range bundle.Profiles {
		existing, _ := corpus.GetProfile(platform)

		if existing != nil && existing.MessageCount >= entry.MessageCount {
			result.Skipped = append(result.Skipped, platform)
			continue
		}

		if err := corpus.SaveProfile(platform, entry.Profile, entry.MessageCount); err != nil {
			return result, fmt.Errorf("saving profile %s: %w", platform, err)
		}

		if existing != nil {
			result.Updated = append(result.Updated, platform)
		} else {
			result.Added = append(result.Added, platform)
		}
	}

	// save persona spec if present and no local one exists
	if bundle.PersonaSpec != "" {
		result.HasPersona = true
	}

	return result, nil
}

// ImportResult summarizes what happened during import.
type ImportResult struct {
	Added      []string
	Updated    []string
	Skipped    []string
	HasPersona bool
}

// ExportPrompt generates a ready-to-paste system prompt that works
// with any AI tool (Claude Code rules, ChatGPT custom instructions,
// Cursor rules, etc.). No tool installation required.
func ExportPrompt(personaPath string) (string, error) {
	combined, err := corpus.GetProfile("combined")
	if err != nil || combined == nil {
		return "", fmt.Errorf("no combined profile found. Run `replicateme ingest` first")
	}

	var b strings.Builder

	b.WriteString("# Writing style replication instructions\n\n")
	b.WriteString("When writing as me (messages, emails, commits, docs, chat), match my actual writing patterns.\n\n")

	// add the persona spec if provided
	if personaPath != "" {
		data, err := os.ReadFile(personaPath)
		if err == nil && len(data) > 0 {
			b.WriteString(string(data))
			b.WriteString("\n\n")
		}
	}

	b.WriteString("## Statistical writing profile\n\n")
	b.WriteString("These numbers come from analyzing my real messages.\n\n")
	b.WriteString(style.StyleProfileToPrompt(combined.Profile))
	b.WriteString("\n\n")

	// add per-platform profiles if they differ meaningfully
	allProfiles, _ := corpus.GetAllProfiles()
	platformProfiles := make([]corpus.PlatformProfile, 0)
	for _, p := range allProfiles {
		if p.Platform != "combined" {
			platformProfiles = append(platformProfiles, p)
		}
	}

	if len(platformProfiles) > 1 {
		b.WriteString("## Per-platform differences\n\n")
		for _, p := range platformProfiles {
			fmt.Fprintf(&b, "### %s (%d messages)\n", p.Platform, p.MessageCount)
			fmt.Fprintf(&b, "- Avg length: %d chars\n", p.Profile.AverageLength)
			fmt.Fprintf(&b, "- Capitalizes first word: %.0f%%\n", p.Profile.CapitalizesFirstWord*100)
			fmt.Fprintf(&b, "- Ends with period: %.0f%%\n", p.Profile.UsesPeriods*100)
			fmt.Fprintf(&b, "- Fragments: %.0f%%\n\n", p.Profile.SentenceFragmentRatio*100)
		}
	}

	b.WriteString("## Rules\n\n")
	b.WriteString("- Match my typical message length for the platform\n")
	b.WriteString("- If I rarely use periods, don't add periods\n")
	b.WriteString("- If I rarely capitalize, don't capitalize\n")
	b.WriteString("- Use my common phrases naturally\n")
	b.WriteString("- Include my typical writing quirks at the appropriate level for the context (casual chat = more quirks, work email = fewer)\n")

	return b.String(), nil
}

// ExportPromptToFile writes the prompt to a file.
func ExportPromptToFile(path string, personaPath string) error {
	prompt, err := ExportPrompt(personaPath)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(prompt), 0644)
}
