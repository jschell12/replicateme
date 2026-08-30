package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jschell12/replicateme/pkg/clipboard"
	"github.com/jschell12/replicateme/pkg/config"
	"github.com/jschell12/replicateme/pkg/connectors"
	"github.com/jschell12/replicateme/pkg/corpus"
	"github.com/jschell12/replicateme/pkg/generate"
	"github.com/jschell12/replicateme/pkg/rag"
	"github.com/jschell12/replicateme/pkg/style"
)

var sources = []string{"imessage", "slack", "gmail", "twitter", "discord", "reddit", "instagram", "github", "tiktok"}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		cmdHelp()
		return
	}

	defer corpus.CloseDB()

	switch args[0] {
	case "ingest":
		cmdIngest(args[1:])
	case "profile":
		cmdProfile(args[1:])
	case "generate", "gen":
		cmdGenerate(args[1:])
	case "config":
		cmdConfig(args[1:])
	case "sources":
		cmdSources()
	case "stats":
		cmdStats()
	case "help":
		cmdHelp()
	default:
		cmdHelp()
	}
}

func cmdHelp() {
	fmt.Printf(`replicateme - learn your writing style, generate messages that sound like you

All data stays on your machine. Bring your own API key.

Commands:
  ingest              Import messages from a data source
    --source SOURCE    Source: %s
    --file PATH        Path to archive ZIP or directory
    --since DATE       Only import messages after this date
    --db-path PATH     Custom path to chat.db (imessage only)
    --user-id ID       User ID to filter (slack)
    --user-name NAME   Display name to filter (slack)
    --email EMAIL      Email to filter (gmail, github)
    --username USER    Username to filter (instagram)
    --repos PATH,...   Comma-separated repo paths (github)

  sources             List available data sources and setup instructions
  stats               Show corpus statistics
  profile             Show your writing style profile
    --platform P       Show profile for a specific platform (or "combined", "all")

  generate (gen)      Generate a message in your style
    --platform P       Platform style (default: from config)
    --quirk N          Quirk level 0-100 (default: from config)
    --variants N       Number of variants (default: 3)
    --persona PATH     Markdown file with persona specification
    --copy [N]         Copy variant N to clipboard (default: 1)
    <context>          The rest of the args are the context/prompt

  config              View or set configuration
    --provider P       LLM provider: anthropic or openai
    --model M          Model name
    --quirk-level N    Default quirk level 0-100
    --platform P       Default platform
    --persona PATH     Default persona spec file
    --enable-quirk Q   Enable a specific quirk toggle
    --disable-quirk Q  Disable a specific quirk toggle
    --rag on/off       Enable/disable RAG vector search for examples
    --qdrant-url URL   Qdrant URL (default: http://10.0.0.2:6333)
    --ollama-url URL   Ollama URL for embeddings (default: http://10.0.0.2:11434)
    --embed-model M    Embedding model (default: bge-m3)

Setup:
  1. export ANTHROPIC_API_KEY=sk-...
  2. replicateme ingest --source imessage
  3. replicateme gen "Friend asks: want to grab dinner tonight?"

Data stored at: %s
`, strings.Join(sources, ", "), config.ConfigDir())
}

func cmdSources() {
	fmt.Println(`Data sources and how to get your data:

imessage (macOS only)
  No file needed - reads ~/Library/Messages/chat.db directly.
  Requires Full Disk Access for your terminal app.
  replicateme ingest --source imessage

slack
  Export your workspace: Workspace Settings > Import/Export > Export.
  replicateme ingest --source slack --file slack-export.zip --user-name "Your Name"

gmail
  Go to takeout.google.com, select Gmail, download as mbox.
  replicateme ingest --source gmail --file Takeout.zip --email you@gmail.com

twitter
  Settings > Your Account > Download an archive of your data.
  replicateme ingest --source twitter --file twitter-archive.zip

discord
  Settings > Privacy & Safety > Request all of my Data.
  replicateme ingest --source discord --file discord-package.zip

reddit
  Settings > Request Your Data (GDPR).
  replicateme ingest --source reddit --file reddit-export.zip

instagram
  Settings > Your Activity > Download Your Information > Request Download.
  replicateme ingest --source instagram --file instagram-data.zip --username yourusername

github
  No archive needed - reads from local git repos.
  replicateme ingest --source github --repos ~/project1,~/project2 --email you@email.com

tiktok
  Settings > Privacy > Download Your Data > Request Data.
  replicateme ingest --source tiktok --file tiktok-data.zip --username yourusername

Ingest as many sources as you want. Messages accumulate in a local database.
Duplicates are automatically skipped.`)
}

func cmdStats() {
	stats, err := corpus.GetCorpusStats()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if stats.TotalMessages == 0 {
		fmt.Println("No messages in corpus. Run `replicateme ingest` first.")
		return
	}

	fmt.Printf("Corpus: %d messages\n\n", stats.TotalMessages)

	fmt.Println("Messages by platform:")
	for _, p := range stats.ByPlatform {
		fmt.Printf("  %-12s %d\n", p.Platform, p.Count)
	}

	if len(stats.Profiles) > 0 {
		fmt.Println("\nStyle profiles:")
		for _, p := range stats.Profiles {
			fmt.Printf("  %-12s %d msgs  (%s)\n", p.Platform, p.MessageCount, p.UpdatedAt)
		}
	}

	fmt.Printf("\nCorpus stored at: %s/corpus.db\n", config.ConfigDir())
}

func cmdIngest(args []string) {
	source := getFlag(args, "--source")
	if source == "" {
		source = "imessage"
	}

	valid := false
	for _, s := range sources {
		if s == source {
			valid = true
			break
		}
	}
	if !valid {
		fmt.Printf("Unknown source %q. Available: %s\n", source, strings.Join(sources, ", "))
		fmt.Println("Run `replicateme sources` for setup instructions.")
		os.Exit(1)
	}

	sinceStr := getFlag(args, "--since")
	var since *time.Time
	if sinceStr != "" {
		t, err := time.Parse("2006-01-02", sinceStr)
		if err != nil {
			t, err = time.Parse(time.RFC3339, sinceStr)
		}
		if err != nil {
			fmt.Printf("Invalid date: %s\n", sinceStr)
			os.Exit(1)
		}
		since = &t
	}

	file := getFlag(args, "--file")

	var messages []corpus.RawMessage
	var err error

	switch source {
	case "imessage":
		fmt.Println("Importing iMessages...")
		messages, err = connectors.ImportIMessages(connectors.IMessageOptions{
			DBPath: getFlag(args, "--db-path"),
			Since:  since,
		})
	case "slack":
		requireFile(file, "slack")
		fmt.Println("Importing Slack messages...")
		messages, err = connectors.ImportSlack(connectors.SlackImportOptions{
			File:     file,
			UserID:   getFlag(args, "--user-id"),
			UserName: getFlag(args, "--user-name"),
			Since:    since,
		})
	case "gmail":
		requireFile(file, "gmail")
		fmt.Println("Importing Gmail...")
		messages, err = connectors.ImportGmail(connectors.GmailImportOptions{
			File:  file,
			Email: getFlag(args, "--email"),
			Since: since,
		})
	case "twitter":
		requireFile(file, "twitter")
		fmt.Println("Importing Twitter/X...")
		messages, err = connectors.ImportTwitter(connectors.TwitterImportOptions{
			File:  file,
			Since: since,
		})
	case "discord":
		requireFile(file, "discord")
		fmt.Println("Importing Discord...")
		messages, err = connectors.ImportDiscord(connectors.DiscordImportOptions{
			File:  file,
			Since: since,
		})
	case "reddit":
		requireFile(file, "reddit")
		fmt.Println("Importing Reddit...")
		messages, err = connectors.ImportReddit(connectors.RedditImportOptions{
			File:  file,
			Since: since,
		})
	case "instagram":
		requireFile(file, "instagram")
		fmt.Println("Importing Instagram...")
		messages, err = connectors.ImportInstagram(connectors.InstagramImportOptions{
			File:     file,
			Username: getFlag(args, "--username"),
			Since:    since,
		})
	case "github":
		reposStr := getFlag(args, "--repos")
		if reposStr == "" {
			fmt.Println("--repos is required for GitHub import.")
			fmt.Println("Example: replicateme ingest --source github --repos ~/project1,~/project2 --email you@email.com")
			os.Exit(1)
		}
		fmt.Println("Importing GitHub commits...")
		repos := strings.Split(reposStr, ",")
		for i := range repos {
			repos[i] = strings.TrimSpace(repos[i])
		}
		messages, err = connectors.ImportGitHub(connectors.GitHubImportOptions{
			Repos: repos,
			Email: getFlag(args, "--email"),
			Since: since,
		})
	case "tiktok":
		requireFile(file, "tiktok")
		fmt.Println("Importing TikTok...")
		messages, err = connectors.ImportTikTok(connectors.TikTokImportOptions{
			File:     file,
			Username: getFlag(args, "--username"),
			Since:    since,
		})
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// filter invalid dates
	filtered := messages[:0]
	for _, m := range messages {
		if !m.Timestamp.IsZero() {
			filtered = append(filtered, m)
		}
	}
	messages = filtered

	fmt.Printf("Parsed %d messages from %s\n", len(messages), source)

	if len(messages) == 0 {
		fmt.Println("No messages found.")
		if source == "imessage" {
			fmt.Println("Make sure Full Disk Access is granted.")
		}
		os.Exit(1)
	}

	// store in corpus
	result, storeErr := corpus.StoreMessages(messages)
	if storeErr != nil {
		fmt.Printf("Error storing messages: %v\n", storeErr)
		os.Exit(1)
	}
	fmt.Printf("Stored %d new messages (%d duplicates skipped)\n", result.Inserted, result.Skipped)
	corpus.LogIngest(source, result.Inserted)

	// build per-platform profile
	platform := platformForSource(source)
	platformMessages, _ := corpus.GetMessages(corpus.GetMessagesOpts{Platform: corpus.Platform(platform)})
	fmt.Printf("\nAnalyzing %s style (%d total messages)...\n", platform, len(platformMessages))
	platformProfile, _ := style.AnalyzeStyle(platformMessages)
	corpus.SaveProfile(platform, platformProfile, len(platformMessages))

	// rebuild combined profile
	allMessages, _ := corpus.GetMessages(corpus.GetMessagesOpts{})
	fmt.Printf("Rebuilding combined profile (%d total messages across all sources)...\n", len(allMessages))
	combinedProfile, _ := style.AnalyzeStyle(allMessages)
	corpus.SaveProfile("combined", combinedProfile, len(allMessages))

	fmt.Printf("\nProfiles saved. Use 'replicateme profile' to view combined, or 'replicateme profile --platform %s' for %s only.\n", platform, platform)

	// index messages in Qdrant for RAG if enabled
	cfg := config.Load()
	if cfg.RAG.Enabled {
		ragCfg := ragConfigFromCfg(cfg)
		if rag.IsAvailable(ragCfg) {
			fmt.Println("\nIndexing messages for RAG search...")
			indexed, err := rag.IndexMessages(ragCfg, messages, func(done, total, skipped int) {
				fmt.Printf("\r  %d/%d embedded (%d already indexed)", done, total, skipped)
			})
			fmt.Println() // newline after progress
			if err != nil {
				fmt.Printf("RAG indexing error: %v\n", err)
			} else {
				fmt.Printf("Indexed %d messages in Qdrant\n", indexed)
			}
		} else {
			fmt.Println("\nRAG is enabled but Qdrant/Ollama not reachable. Skipping indexing.")
		}
	}

	fmt.Printf("\n--- %s style ---\n\n", platform)
	fmt.Println(style.StyleProfileToPrompt(platformProfile))
}

func cmdProfile(args []string) {
	platform := getFlag(args, "--platform")
	if platform == "" {
		platform = "combined"
	}

	if platform == "all" {
		profiles, err := corpus.GetAllProfiles()
		if err != nil || len(profiles) == 0 {
			fmt.Println("No profiles found. Run `replicateme ingest` first.")
			os.Exit(1)
		}
		for _, p := range profiles {
			fmt.Printf("\n=== %s (%d messages) ===\n\n", p.Platform, p.MessageCount)
			fmt.Println(style.StyleProfileToPrompt(p.Profile))
		}
		return
	}

	result, err := corpus.GetProfile(platform)
	if err != nil || result == nil {
		fmt.Printf("No profile found for %q. Run `replicateme ingest` first.\n", platform)
		profiles, _ := corpus.GetAllProfiles()
		if len(profiles) > 0 {
			names := make([]string, len(profiles))
			for i, p := range profiles {
				names[i] = p.Platform
			}
			fmt.Printf("Available profiles: %s\n", strings.Join(names, ", "))
		}
		os.Exit(1)
	}

	fmt.Printf("%s profile (%d messages):\n\n", platform, result.MessageCount)
	fmt.Println(style.StyleProfileToPrompt(result.Profile))
}

func cmdGenerate(args []string) {
	cfg := config.Load()

	platform := getFlag(args, "--platform")
	if platform == "" {
		platform = cfg.DefaultPlatform
	}

	quirkLevel := cfg.QuirkLevel
	if q := getFlag(args, "--quirk"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			quirkLevel = v
		}
	}

	variants := 3
	if v := getFlag(args, "--variants"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			variants = n
		}
	}

	// persona spec: --persona flag overrides config default
	personaPath := getFlag(args, "--persona")
	if personaPath == "" {
		personaPath = cfg.Persona
	}
	var personaSpec string
	if personaPath != "" {
		data, err := os.ReadFile(personaPath)
		if err != nil {
			fmt.Printf("Error reading persona file: %v\n", err)
			os.Exit(1)
		}
		personaSpec = string(data)
	}

	// clipboard copy: --copy with optional variant number
	copyVariant := -1 // -1 means no copy
	if hasFlag(args, "--copy") {
		copyVariant = 1 // default to variant 1
		if v := getFlag(args, "--copy"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				copyVariant = n
			}
		}
	}

	var contextParts []string
	skipNext := false
	for _, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(a, "--") {
			skipNext = true
			continue
		}
		contextParts = append(contextParts, a)
	}
	context := strings.Join(contextParts, " ")

	if context == "" {
		fmt.Println("Usage: replicateme gen [--platform P] [--quirk N] [--persona PATH] [--copy [N]] <context>")
		fmt.Println(`Example: replicateme gen "Friend asks: want to grab dinner?"`)
		os.Exit(1)
	}

	platformProfile, _ := corpus.GetProfile(platform)
	combinedProfile, _ := corpus.GetProfile("combined")
	profileResult := platformProfile
	if profileResult == nil {
		profileResult = combinedProfile
	}
	if profileResult == nil {
		fmt.Println("No profile found. Run `replicateme ingest` first.")
		os.Exit(1)
	}

	fmt.Printf("Platform: %s | Quirk: %d%% | Variants: %d\n", platform, quirkLevel, variants)
	if platformProfile != nil {
		fmt.Printf("Using %s profile (%d messages)\n", platform, platformProfile.MessageCount)
	} else {
		fmt.Printf("No %s profile found, using combined profile\n", platform)
	}
	fmt.Printf("Context: %s\n\n", context)

	// try RAG search for similar examples, fall back to random
	var examples []corpus.RawMessage
	if cfg.RAG.Enabled {
		ragCfg := ragConfigFromCfg(cfg)
		if rag.IsAvailable(ragCfg) {
			ragExamples, err := rag.Search(ragCfg, context, platform, 15)
			if err == nil && len(ragExamples) > 0 {
				examples = ragExamples
				fmt.Printf("Found %d similar examples via RAG\n\n", len(examples))
			} else {
				if err != nil {
					fmt.Printf("RAG search failed (%v), falling back to random examples\n", err)
				}
			}
		}
	}
	if len(examples) == 0 {
		examples, _ = corpus.GetExamples(corpus.Platform(platform), 20)
	}

	// convert config quirk toggles to generate package type
	qt := generate.QuirkToggles{
		Misspellings:       cfg.Quirks.Misspellings,
		GrammarErrors:      cfg.Quirks.GrammarErrors,
		MissingApostrophes: cfg.Quirks.MissingApostrophes,
		LowercaseI:         cfg.Quirks.LowercaseI,
		SkipPunctuation:    cfg.Quirks.SkipPunctuation,
		DoubleSpaces:       cfg.Quirks.DoubleSpaces,
		Fragments:          cfg.Quirks.Fragments,
	}

	results, err := generate.GenerateMessage(generate.GenerateRequest{
		Platform:        platform,
		Context:         context,
		QuirkLevel:      quirkLevel,
		SimilarMessages: examples,
		StyleProfile:    profileResult.Profile,
		Variants:        variants,
		PersonaSpec:     personaSpec,
		QuirkToggles:    qt,
		Config: generate.Config{
			Provider: cfg.Provider,
			Model:    cfg.Model,
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for i, r := range results {
		fmt.Printf("[%d] %s\n", i+1, strings.TrimSpace(r))
	}

	// copy to clipboard if requested
	if copyVariant > 0 && len(results) > 0 {
		idx := copyVariant - 1
		if idx >= len(results) {
			idx = 0
		}
		text := strings.TrimSpace(results[idx])
		if err := clipboard.Copy(text); err != nil {
			fmt.Fprintf(os.Stderr, "Could not copy to clipboard: %v\n", err)
		} else {
			fmt.Printf("\nVariant %d copied to clipboard.\n", idx+1)
		}
	}
}

func cmdConfig(args []string) {
	cfg := config.Load()

	provider := getFlag(args, "--provider")
	model := getFlag(args, "--model")
	quirkLevelStr := getFlag(args, "--quirk-level")
	platform := getFlag(args, "--platform")
	persona := getFlag(args, "--persona")
	enableQuirk := getFlag(args, "--enable-quirk")
	disableQuirk := getFlag(args, "--disable-quirk")
	ragEnabled := getFlag(args, "--rag")
	qdrantURL := getFlag(args, "--qdrant-url")
	ollamaURL := getFlag(args, "--ollama-url")
	embedModel := getFlag(args, "--embed-model")

	hasChanges := provider != "" || model != "" || quirkLevelStr != "" || platform != "" || persona != "" || enableQuirk != "" || disableQuirk != "" || ragEnabled != "" || qdrantURL != "" || ollamaURL != "" || embedModel != ""

	if provider != "" {
		cfg.Provider = provider
	}
	if model != "" {
		cfg.Model = model
	}
	if quirkLevelStr != "" {
		if v, err := strconv.Atoi(quirkLevelStr); err == nil {
			cfg.QuirkLevel = v
		}
	}
	if platform != "" {
		cfg.DefaultPlatform = platform
	}
	if persona != "" {
		cfg.Persona = persona
	}
	if enableQuirk != "" {
		setQuirkToggle(&cfg.Quirks, enableQuirk, true)
	}
	if disableQuirk != "" {
		setQuirkToggle(&cfg.Quirks, disableQuirk, false)
	}
	if ragEnabled != "" {
		cfg.RAG.Enabled = ragEnabled == "true" || ragEnabled == "on" || ragEnabled == "1"
	}
	if qdrantURL != "" {
		cfg.RAG.QdrantURL = qdrantURL
	}
	if ollamaURL != "" {
		cfg.RAG.OllamaURL = ollamaURL
	}
	if embedModel != "" {
		cfg.RAG.EmbedModel = embedModel
	}

	if hasChanges {
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config updated:")
	} else {
		fmt.Println("Current config:")
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(data))
	fmt.Printf("\nConfig file: %s/config.json\n", config.ConfigDir())
}

func setQuirkToggle(q *config.QuirkToggles, name string, val bool) {
	v := val
	switch strings.ToLower(name) {
	case "misspellings":
		q.Misspellings = &v
	case "grammarerrors":
		q.GrammarErrors = &v
	case "missingapostrophes":
		q.MissingApostrophes = &v
	case "lowercasei":
		q.LowercaseI = &v
	case "skippunctuation":
		q.SkipPunctuation = &v
	case "doublespaces":
		q.DoubleSpaces = &v
	case "fragments":
		q.Fragments = &v
	default:
		fmt.Printf("Unknown quirk %q. Available: misspellings, grammarErrors, missingApostrophes, lowercaseI, skipPunctuation, doubleSpaces, fragments\n", name)
		os.Exit(1)
	}
}

func getFlag(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// hasFlag checks if a flag is present in args (with or without a value).
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func requireFile(file, source string) {
	if file == "" {
		fmt.Printf("--file is required for %s import.\n", source)
		fmt.Printf("Example: replicateme ingest --source %s --file %s-export.zip\n", source, source)
		os.Exit(1)
	}
}

func platformForSource(source string) string {
	if source == "gmail" {
		return "email"
	}
	return source
}

func ragConfigFromCfg(cfg config.Config) rag.RAGConfig {
	rc := rag.DefaultConfig()
	if cfg.RAG.QdrantURL != "" {
		rc.QdrantURL = cfg.RAG.QdrantURL
	}
	if cfg.RAG.OllamaURL != "" {
		rc.OllamaURL = cfg.RAG.OllamaURL
	}
	if cfg.RAG.EmbedModel != "" {
		rc.EmbedModel = cfg.RAG.EmbedModel
	}
	return rc
}
