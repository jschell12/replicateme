package generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/jschell12/replicateme/pkg/corpus"
)

type Config struct {
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"` // custom endpoint for openai-compatible providers
}

type GenerateRequest struct {
	Platform        string
	Context         string
	Instruction     string
	QuirkLevel      int
	SimilarMessages []corpus.RawMessage
	StyleProfile    corpus.StyleProfile
	Variants        int
	Config          Config
	PersonaSpec     string
	QuirkToggles    QuirkToggles
}

func GenerateMessage(req GenerateRequest) ([]string, error) {
	if req.Variants == 0 {
		req.Variants = 3
	}

	opts := GenerateOptions{
		Platform:        req.Platform,
		Context:         req.Context,
		SimilarMessages: req.SimilarMessages,
		StyleProfile:    req.StyleProfile,
		QuirkLevel:      req.QuirkLevel,
		Instruction:     req.Instruction,
		PersonaSpec:     req.PersonaSpec,
		QuirkToggles:    req.QuirkToggles,
	}

	system := BuildSystemPrompt(opts)
	user := BuildUserPrompt(opts)

	switch req.Config.Provider {
	case "openai":
		return generateOpenAI(system, user, req.Variants, req.Config)
	case "claude-cli":
		return generateClaudeCLI(system, user, req.Variants)
	case "ollama":
		return generateOllama(system, user, req.Variants, req.Config)
	default:
		return generateAnthropic(system, user, req.Variants, req.Config)
	}
}

func generateAnthropic(system, user string, variants int, cfg Config) ([]string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6-20250725"
	}

	results := make([]string, 0, variants)

	for i := 0; i < variants; i++ {
		temp := 0.8 + float64(i)*0.05

		body := map[string]any{
			"model":       model,
			"max_tokens":  256,
			"temperature": temp,
			"system":      system,
			"messages": []map[string]string{
				{"role": "user", "content": user},
			},
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("%d %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		if len(result.Content) > 0 && result.Content[0].Type == "text" {
			results = append(results, result.Content[0].Text)
		}
	}

	return results, nil
}

func generateOpenAI(system, user string, variants int, cfg Config) ([]string, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	// only require API key for the real OpenAI endpoint
	if apiKey == "" && baseURL == "https://api.openai.com" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o"
	}

	results := make([]string, 0, variants)

	for i := 0; i < variants; i++ {
		temp := 0.8 + float64(i)*0.05

		body := map[string]any{
			"model":       model,
			"max_tokens":  256,
			"temperature": temp,
			"messages": []map[string]string{
				{"role": "system", "content": system},
				{"role": "user", "content": user},
			},
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("%d %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		if len(result.Choices) > 0 {
			results = append(results, result.Choices[0].Message.Content)
		}
	}

	return results, nil
}

func generateClaudeCLI(system, user string, variants int) ([]string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH. Install it or use a different provider")
	}

	results := make([]string, 0, variants)

	prompt := fmt.Sprintf("System instructions:\n%s\n\nUser request:\n%s\n\nRespond with ONLY the message text. No quotes, no explanation.", system, user)

	for i := 0; i < variants; i++ {
		cmd := exec.Command(claudePath, "-p", prompt)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("claude CLI failed: %v\n%s", err, stderr.String())
		}

		text := strings.TrimSpace(stdout.String())
		if text != "" {
			results = append(results, text)
		}
	}

	return results, nil
}

func generateOllama(system, user string, variants int, cfg Config) ([]string, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://10.0.0.2:11434"
	}

	model := cfg.Model
	if model == "" {
		model = "mistral-small:24b"
	}

	results := make([]string, 0, variants)

	for i := 0; i < variants; i++ {
		temp := 0.8 + float64(i)*0.05

		body := map[string]any{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": system},
				{"role": "user", "content": user},
			},
			"stream": false,
			"options": map[string]any{
				"temperature": temp,
				"num_predict": 256,
			},
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		resp, err := http.Post(baseURL+"/api/chat", "application/json", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("ollama unreachable at %s: %w", baseURL, err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("%d %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		text := strings.TrimSpace(result.Message.Content)
		if text != "" {
			results = append(results, text)
		}
	}

	return results, nil
}
