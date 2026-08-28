package generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jschell12/replicateme/pkg/corpus"
)

type Config struct {
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
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
	}

	system := BuildSystemPrompt(opts)
	user := BuildUserPrompt(opts)

	if req.Config.Provider == "openai" {
		return generateOpenAI(system, user, req.Variants, req.Config)
	}
	return generateAnthropic(system, user, req.Variants, req.Config)
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
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
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

		req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

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
