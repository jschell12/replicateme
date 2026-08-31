package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jschell12/replicateme/pkg/config"
	"github.com/jschell12/replicateme/pkg/connectors"
	"github.com/jschell12/replicateme/pkg/corpus"
	"github.com/jschell12/replicateme/pkg/generate"
	"github.com/jschell12/replicateme/pkg/rag"
	"github.com/jschell12/replicateme/pkg/style"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetModels fetches available models from the configured provider's API.
func (a *App) GetModels(provider string, baseURL string) ([]string, error) {
	switch provider {
	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		req, _ := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var result struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		models := make([]string, len(result.Data))
		for i, m := range result.Data {
			models[i] = m.ID
		}
		return models, nil

	case "openai":
		url := "https://api.openai.com"
		if baseURL != "" {
			url = baseURL
		}
		apiKey := os.Getenv("OPENAI_API_KEY")
		req, _ := http.NewRequest("GET", url+"/v1/models", nil)
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var result struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		models := make([]string, len(result.Data))
		for i, m := range result.Data {
			models[i] = m.ID
		}
		return models, nil

	case "ollama":
		url := "http://10.0.0.2:11434"
		if baseURL != "" {
			url = baseURL
		}
		resp, err := httpClient.Get(url + "/api/tags")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var result struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		models := make([]string, len(result.Models))
		for i, m := range result.Models {
			models[i] = m.Name
		}
		return models, nil

	case "claude-cli":
		return []string{"(uses CLI default)"}, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// GetConfig returns the current configuration.
func (a *App) GetConfig() config.Config {
	return config.Load()
}

// SaveConfig persists the given configuration.
func (a *App) SaveConfig(cfg config.Config) error {
	return config.Save(cfg)
}

// GetCorpusStats returns aggregate corpus statistics.
func (a *App) GetCorpusStats() (*corpus.CorpusStats, error) {
	return corpus.GetCorpusStats()
}

// GetProfile retrieves a single platform profile (or "combined").
func (a *App) GetProfile(platform string) (*corpus.ProfileResult, error) {
	return corpus.GetProfile(platform)
}

// GetAllProfiles returns all stored style profiles.
func (a *App) GetAllProfiles() ([]corpus.PlatformProfile, error) {
	return corpus.GetAllProfiles()
}

// IngestResult is the result of an ingest operation.
type IngestResult struct {
	MessageCount int    `json:"messageCount"`
	NewCount     int    `json:"newCount"`
	ProfileNote  string `json:"profileNote"`
}

// IngestSource ingests messages from a data source, stores them, analyzes style,
// and optionally indexes into RAG.
func (a *App) IngestSource(source string, file string, options map[string]string) (IngestResult, error) {
	var messages []corpus.RawMessage
	var err error

	switch source {
	case "imessage":
		messages, err = connectors.ImportIMessages(connectors.IMessageOptions{})
	case "slack":
		opts := connectors.SlackImportOptions{File: file}
		if v, ok := options["userName"]; ok {
			opts.UserName = v
		}
		messages, err = connectors.ImportSlack(opts)
	case "gmail":
		opts := connectors.GmailImportOptions{File: file}
		if v, ok := options["email"]; ok {
			opts.Email = v
		}
		messages, err = connectors.ImportGmail(opts)
	case "twitter":
		messages, err = connectors.ImportTwitter(connectors.TwitterImportOptions{File: file})
	case "discord":
		messages, err = connectors.ImportDiscord(connectors.DiscordImportOptions{File: file})
	case "reddit":
		messages, err = connectors.ImportReddit(connectors.RedditImportOptions{File: file})
	case "instagram":
		opts := connectors.InstagramImportOptions{File: file}
		if v, ok := options["username"]; ok {
			opts.Username = v
		}
		messages, err = connectors.ImportInstagram(opts)
	case "github":
		opts := connectors.GitHubImportOptions{}
		if v, ok := options["repos"]; ok {
			opts.Repos = strings.Split(v, ",")
			for i := range opts.Repos {
				opts.Repos[i] = strings.TrimSpace(opts.Repos[i])
			}
		}
		if v, ok := options["email"]; ok {
			opts.Email = v
		}
		messages, err = connectors.ImportGitHub(opts)
	case "tiktok":
		messages, err = connectors.ImportTikTok(connectors.TikTokImportOptions{File: file})
	default:
		return IngestResult{}, fmt.Errorf("unknown source: %s", source)
	}

	if err != nil {
		return IngestResult{}, err
	}

	if len(messages) == 0 {
		return IngestResult{MessageCount: 0, NewCount: 0, ProfileNote: "No messages found"}, nil
	}

	// Store messages
	storeResult, err := corpus.StoreMessages(messages)
	if err != nil {
		return IngestResult{}, fmt.Errorf("store messages: %w", err)
	}

	// Log the ingest
	_ = corpus.LogIngest(source, storeResult.Inserted)

	// Analyze style and save profiles
	profileNote := ""
	if len(messages) >= 5 {
		// Determine platform from messages
		platform := string(messages[0].Platform)
		prof, analyzeErr := style.AnalyzeStyle(messages)
		if analyzeErr == nil {
			_ = corpus.SaveProfile(platform, prof, len(messages))
			profileNote = fmt.Sprintf("Style profile updated for %s", platform)
		}

		// Also update combined profile with all messages
		allMsgs, _ := corpus.GetMessages(corpus.GetMessagesOpts{Limit: 10000})
		if len(allMsgs) > 0 {
			combinedProf, combinedErr := style.AnalyzeStyle(allMsgs)
			if combinedErr == nil {
				_ = corpus.SaveProfile("combined", combinedProf, len(allMsgs))
			}
		}
	}

	// Index into RAG if enabled
	cfg := config.Load()
	if cfg.RAG.Enabled {
		ragCfg := rag.RAGConfig{
			QdrantURL:  cfg.RAG.QdrantURL,
			OllamaURL:  cfg.RAG.OllamaURL,
			EmbedModel: cfg.RAG.EmbedModel,
		}
		if rag.IsAvailable(ragCfg) {
			_, _ = rag.IndexMessages(ragCfg, messages)
		}
	}

	return IngestResult{
		MessageCount: len(messages),
		NewCount:     storeResult.Inserted,
		ProfileNote:  profileNote,
	}, nil
}

// Generate produces message variants in the user's style.
func (a *App) Generate(platform string, contextText string, quirkLevel int, personaPath string, variants int) ([]string, error) {
	if variants <= 0 {
		variants = 3
	}

	cfg := config.Load()

	if platform == "" {
		platform = cfg.DefaultPlatform
	}
	if platform == "" {
		platform = "imessage"
	}

	// Load profile
	prof, err := corpus.GetProfile(platform)
	if prof == nil {
		prof, err = corpus.GetProfile("combined")
	}
	if err != nil || prof == nil {
		return nil, fmt.Errorf("no style profile found. Ingest some messages first")
	}

	// Get example messages
	var examples []corpus.RawMessage

	if cfg.RAG.Enabled {
		ragCfg := rag.RAGConfig{
			QdrantURL:  cfg.RAG.QdrantURL,
			OllamaURL:  cfg.RAG.OllamaURL,
			EmbedModel: cfg.RAG.EmbedModel,
		}
		if rag.IsAvailable(ragCfg) {
			examples, _ = rag.Search(ragCfg, contextText, platform, 20)
		}
	}

	if len(examples) == 0 {
		examples, _ = corpus.GetExamples(corpus.Platform(platform), 20)
	}

	// Load persona spec
	var personaSpec string
	if personaPath != "" {
		data, readErr := os.ReadFile(personaPath)
		if readErr == nil {
			personaSpec = string(data)
		}
	}

	// Map config quirk toggles to generate quirk toggles
	qt := generate.QuirkToggles{
		Misspellings:       cfg.Quirks.Misspellings,
		GrammarErrors:      cfg.Quirks.GrammarErrors,
		MissingApostrophes: cfg.Quirks.MissingApostrophes,
		LowercaseI:         cfg.Quirks.LowercaseI,
		SkipPunctuation:    cfg.Quirks.SkipPunctuation,
		DoubleSpaces:       cfg.Quirks.DoubleSpaces,
		Fragments:          cfg.Quirks.Fragments,
	}

	req := generate.GenerateRequest{
		Platform:        platform,
		Context:         contextText,
		QuirkLevel:      quirkLevel,
		SimilarMessages: examples,
		StyleProfile:    prof.Profile,
		Variants:        variants,
		PersonaSpec:     personaSpec,
		QuirkToggles:    qt,
		Config: generate.Config{
			Provider: cfg.Provider,
			Model:    cfg.Model,
			BaseURL:  cfg.BaseURL,
		},
	}

	return generate.GenerateMessage(req)
}

// CopyToClipboard copies text to the system clipboard.
func (a *App) CopyToClipboard(text string) error {
	wailsRuntime.ClipboardSetText(a.ctx, text)
	return nil
}

// SourceInfo describes an available data source.
type SourceInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	RequiresFile bool   `json:"requiresFile"`
}

// GetSources returns the list of available data sources.
func (a *App) GetSources() []SourceInfo {
	return []SourceInfo{
		{Name: "imessage", Description: "iMessage (reads local chat.db)", RequiresFile: false},
		{Name: "slack", Description: "Slack workspace export (ZIP or directory)", RequiresFile: true},
		{Name: "gmail", Description: "Gmail/mbox export (ZIP or .mbox file)", RequiresFile: true},
		{Name: "twitter", Description: "Twitter/X data archive (ZIP or directory)", RequiresFile: true},
		{Name: "discord", Description: "Discord data export (ZIP or directory)", RequiresFile: true},
		{Name: "reddit", Description: "Reddit data archive (ZIP or directory)", RequiresFile: true},
		{Name: "instagram", Description: "Instagram data download (ZIP or directory)", RequiresFile: true},
		{Name: "github", Description: "GitHub commits (local git repos)", RequiresFile: false},
		{Name: "tiktok", Description: "TikTok data download (ZIP or directory)", RequiresFile: true},
	}
}

// SelectFile opens a native file dialog and returns the selected file path.
func (a *App) SelectFile() (string, error) {
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select File",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Archives & Data", Pattern: "*.zip;*.mbox;*.json;*.csv"},
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// IsRAGAvailable checks if RAG infrastructure is reachable.
func (a *App) IsRAGAvailable() bool {
	cfg := config.Load()
	if !cfg.RAG.Enabled {
		return false
	}
	ragCfg := rag.RAGConfig{
		QdrantURL:  cfg.RAG.QdrantURL,
		OllamaURL:  cfg.RAG.OllamaURL,
		EmbedModel: cfg.RAG.EmbedModel,
	}
	return rag.IsAvailable(ragCfg)
}
