package style

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/jschell12/replicateme/pkg/corpus"
)

// AnalyzeStyle computes a StyleProfile from a slice of messages.
func AnalyzeStyle(messages []corpus.RawMessage) (corpus.StyleProfile, error) {
	if len(messages) == 0 {
		return corpus.StyleProfile{}, fmt.Errorf("no messages to analyze")
	}

	texts := make([]string, len(messages))
	for i, m := range messages {
		texts[i] = m.Text
	}

	n := float64(len(texts))

	// average length
	totalLen := 0
	for _, t := range texts {
		totalLen += len(t)
	}
	avgLength := float64(totalLen) / n

	// capitalizes first word
	capsFirst := 0
	capsFirstRe := regexp.MustCompile(`^[A-Z]`)
	for _, t := range texts {
		if capsFirstRe.MatchString(t) {
			capsFirst++
		}
	}

	// contractions
	contractionsRe := regexp.MustCompile(`(?i)\b(i'm|i'll|i've|it's|that's|there's|don't|can't|won't|didn't|isn't|aren't|wasn't|weren't|hasn't|haven't|couldn't|wouldn't|shouldn't|y'all|let's|he's|she's|we're|they're|you're|who's|what's|where's|how's)\b`)
	withContractions := 0
	for _, t := range texts {
		if contractionsRe.MatchString(t) {
			withContractions++
		}
	}

	// punctuation
	endsWithPeriod := regexp.MustCompile(`\.\s*$`)
	withPeriods := 0
	withCommas := 0
	withExclamation := 0
	withQuestion := 0
	for _, t := range texts {
		if endsWithPeriod.MatchString(t) {
			withPeriods++
		}
		if strings.Contains(t, ",") {
			withCommas++
		}
		if strings.Contains(t, "!") {
			withExclamation++
		}
		if strings.Contains(t, "?") {
			withQuestion++
		}
	}

	// emoji
	emojiRe := regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F1E0}-\x{1F1FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}]`)
	withEmoji := 0
	for _, t := range texts {
		if emojiRe.MatchString(t) {
			withEmoji++
		}
	}

	// sentence fragments: < 30 chars and <= 5 words
	fragmentish := 0
	for _, t := range texts {
		if len(t) < 30 {
			words := strings.Fields(t)
			if len(words) <= 5 {
				fragmentish++
			}
		}
	}

	// lowercase "i" as standalone word (not I'm, I'll, etc.)
	// Go regexp doesn't support lookahead, so match \bi\b then check the next char isn't '
	lowercaseIRe := regexp.MustCompile(`\bi\b`)
	iAnyRe := regexp.MustCompile(`(?i)\b[iI]\b`)
	lowercaseI := 0
	iTotal := 0
	for _, t := range texts {
		if loc := lowercaseIRe.FindStringIndex(t); loc != nil {
			// check char after match isn't apostrophe
			end := loc[1]
			nextChar := string([]rune(t)[end:])
			if len(nextChar) == 0 || (nextChar[0] != '\'') {
				lowercaseI++
			}
		}
		if iAnyRe.MatchString(t) {
			iTotal++
		}
	}

	var lowercaseIRatio float64
	if iTotal > 0 {
		lowercaseIRatio = float64(lowercaseI) / float64(iTotal)
	}

	commonPhrases := findCommonPhrases(texts, 5)
	typicalErrors := findTypicalErrors(texts)

	return corpus.StyleProfile{
		AverageLength:         int(math.Round(avgLength)),
		CapitalizesFirstWord:  round3(float64(capsFirst) / n),
		UsesContractions:      round3(float64(withContractions) / n),
		UsesPeriods:           round3(float64(withPeriods) / n),
		UsesCommas:            round3(float64(withCommas) / n),
		UsesExclamation:       round3(float64(withExclamation) / n),
		UsesQuestionMark:      round3(float64(withQuestion) / n),
		UsesEmoji:             round3(float64(withEmoji) / n),
		CommonPhrases:         commonPhrases,
		TypicalErrors:         typicalErrors,
		SentenceFragmentRatio: round3(float64(fragmentish) / n),
		LowercaseIRatio:       round3(lowercaseIRatio),
	}, nil
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

func findCommonPhrases(texts []string, minCount int) []string {
	ngrams := make(map[string]int)

	for _, text := range texts {
		words := strings.Fields(strings.ToLower(text))
		for ngramLen := 2; ngramLen <= 3; ngramLen++ {
			for i := 0; i <= len(words)-ngramLen; i++ {
				gram := strings.Join(words[i:i+ngramLen], " ")
				ngrams[gram]++
			}
		}
	}

	type entry struct {
		phrase string
		count  int
	}
	var entries []entry
	for phrase, count := range ngrams {
		if count >= minCount {
			entries = append(entries, entry{phrase, count})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	limit := 30
	if len(entries) < limit {
		limit = len(entries)
	}
	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = entries[i].phrase
	}
	return result
}

func findTypicalErrors(texts []string) []corpus.TypicalError {
	var errors []corpus.TypicalError

	// missing apostrophes
	missingApostrophe := []struct {
		re    *regexp.Regexp
		label string
	}{
		{regexp.MustCompile(`(?i)\bdont\b`), "dont (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\bcant\b`), "cant (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\bwont\b`), "wont (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\bdidnt\b`), "didnt (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\bisnt\b`), "isnt (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\bthats\b`), "thats (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\btheres\b`), "theres (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\bits\b`), "its (possessive vs contraction)"},
		{regexp.MustCompile(`(?i)\bim\b`), "im (missing apostrophe)"},
		{regexp.MustCompile(`(?i)\byoure\b`), "youre (missing apostrophe)"},
	}

	for _, ma := range missingApostrophe {
		var matches []string
		for _, t := range texts {
			if ma.re.MatchString(t) {
				matches = append(matches, t)
			}
		}
		if len(matches) >= 3 {
			examples := matches
			if len(examples) > 3 {
				examples = examples[:3]
			}
			trimmed := make([]string, len(examples))
			for i, ex := range examples {
				if len(ex) > 80 {
					trimmed[i] = ex[:80]
				} else {
					trimmed[i] = ex
				}
			}
			errors = append(errors, corpus.TypicalError{
				Pattern:   ma.label,
				Frequency: len(matches),
				Examples:  trimmed,
			})
		}
	}

	// double spaces
	doubleSpaceRe := regexp.MustCompile(`  `)
	var doubleSpaces []string
	for _, t := range texts {
		if doubleSpaceRe.MatchString(t) {
			doubleSpaces = append(doubleSpaces, t)
		}
	}
	if len(doubleSpaces) >= 3 {
		examples := doubleSpaces
		if len(examples) > 3 {
			examples = examples[:3]
		}
		trimmed := make([]string, len(examples))
		for i, ex := range examples {
			if len(ex) > 80 {
				trimmed[i] = ex[:80]
			} else {
				trimmed[i] = ex
			}
		}
		errors = append(errors, corpus.TypicalError{
			Pattern:   "double spaces",
			Frequency: len(doubleSpaces),
			Examples:  trimmed,
		})
	}

	// repeated words (Go RE2 doesn't support backreferences, check manually)
	var repeatedWords []string
	for _, t := range texts {
		words := strings.Fields(strings.ToLower(t))
		for i := 0; i+1 < len(words); i++ {
			if words[i] == words[i+1] && len(words[i]) > 1 {
				repeatedWords = append(repeatedWords, t)
				break
			}
		}
	}
	if len(repeatedWords) >= 2 {
		examples := repeatedWords
		if len(examples) > 3 {
			examples = examples[:3]
		}
		trimmed := make([]string, len(examples))
		for i, ex := range examples {
			if len(ex) > 80 {
				trimmed[i] = ex[:80]
			} else {
				trimmed[i] = ex
			}
		}
		errors = append(errors, corpus.TypicalError{
			Pattern:   "repeated words",
			Frequency: len(repeatedWords),
			Examples:  trimmed,
		})
	}

	// sort by frequency descending
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Frequency > errors[j].Frequency
	})

	return errors
}

// StyleProfileToPrompt renders a profile as a human-readable prompt section.
func StyleProfileToPrompt(profile corpus.StyleProfile) string {
	var sb strings.Builder

	sb.WriteString("## Writing style characteristics\n\n")
	sb.WriteString(fmt.Sprintf("- Average message length: %d characters\n", profile.AverageLength))
	sb.WriteString(fmt.Sprintf("- Capitalizes first word: %s of the time\n", pct(profile.CapitalizesFirstWord)))
	sb.WriteString(fmt.Sprintf("- Uses contractions: %s of the time\n", pct(profile.UsesContractions)))
	sb.WriteString(fmt.Sprintf("- Ends with period: %s of the time\n", pct(profile.UsesPeriods)))
	sb.WriteString(fmt.Sprintf("- Uses commas: %s of the time\n", pct(profile.UsesCommas)))
	sb.WriteString(fmt.Sprintf("- Uses exclamation marks: %s of the time\n", pct(profile.UsesExclamation)))
	sb.WriteString(fmt.Sprintf("- Uses question marks: %s of the time\n", pct(profile.UsesQuestionMark)))
	sb.WriteString(fmt.Sprintf("- Uses emoji: %s of the time\n", pct(profile.UsesEmoji)))
	sb.WriteString(fmt.Sprintf("- Short fragment messages: %s of the time\n", pct(profile.SentenceFragmentRatio)))

	if profile.LowercaseIRatio > 0.1 {
		sb.WriteString(fmt.Sprintf("- Uses lowercase \"i\" instead of \"I\": %s of the time\n", pct(profile.LowercaseIRatio)))
	}

	if len(profile.CommonPhrases) > 0 {
		sb.WriteString("\n## Common phrases\n")
		limit := 15
		if len(profile.CommonPhrases) < limit {
			limit = len(profile.CommonPhrases)
		}
		for _, p := range profile.CommonPhrases[:limit] {
			sb.WriteString(fmt.Sprintf("- \"%s\"\n", p))
		}
	}

	if len(profile.TypicalErrors) > 0 {
		sb.WriteString("\n## Typical writing quirks/errors\n")
		for _, err := range profile.TypicalErrors {
			quoted := make([]string, len(err.Examples))
			for i, ex := range err.Examples {
				quoted[i] = fmt.Sprintf("\"%s\"", ex)
			}
			sb.WriteString(fmt.Sprintf("- %s (%d occurrences). Examples: %s\n",
				err.Pattern, err.Frequency, strings.Join(quoted, ", ")))
		}
	}

	return sb.String()
}

func pct(ratio float64) string {
	return fmt.Sprintf("%d%%", int(math.Round(ratio*100)))
}
