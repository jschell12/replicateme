package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jschell12/replicateme/pkg/corpus"
)

const (
	collectionName = "replicateme_messages"
	embeddingDim   = 1024
)

type RAGConfig struct {
	QdrantURL   string // e.g. http://10.0.0.2:6333
	OllamaURL   string // e.g. http://10.0.0.2:11434
	EmbedModel  string // e.g. bge-m3
}

func DefaultConfig() RAGConfig {
	return RAGConfig{
		QdrantURL:  "http://10.0.0.2:6333",
		OllamaURL:  "http://10.0.0.2:11434",
		EmbedModel: "bge-m3",
	}
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// EnsureCollection creates the Qdrant collection if it doesn't exist.
func EnsureCollection(cfg RAGConfig) error {
	url := fmt.Sprintf("%s/collections/%s", cfg.QdrantURL, collectionName)

	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("qdrant unreachable: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil // already exists
	}

	body := map[string]any{
		"vectors": map[string]any{
			"size":     embeddingDim,
			"distance": "Cosine",
		},
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %d %s", resp.StatusCode, string(b))
	}

	return nil
}

// Embed returns the embedding vector for the given text.
func Embed(cfg RAGConfig, text string) ([]float64, error) {
	body := map[string]string{
		"model":  cfg.EmbedModel,
		"prompt": text,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := httpClient.Post(cfg.OllamaURL+"/api/embeddings", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ollama unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Embedding, nil
}

// ProgressFunc is called during indexing with (indexed, total, skipped).
type ProgressFunc func(indexed, total, skipped int)

// IndexMessages embeds and upserts messages into Qdrant. Returns count of newly indexed.
func IndexMessages(cfg RAGConfig, messages []corpus.RawMessage, progress ...ProgressFunc) (int, error) {
	var onProgress ProgressFunc
	if len(progress) > 0 {
		onProgress = progress[0]
	}
	if err := EnsureCollection(cfg); err != nil {
		return 0, err
	}

	// check which IDs already exist
	existing, err := getExistingIDs(cfg, messages)
	if err != nil {
		return 0, err
	}

	var points []map[string]any
	indexed := 0
	skipped := 0
	total := len(messages)

	for i, msg := range messages {
		if existing[msg.ID] {
			skipped++
			if onProgress != nil && (i+1)%100 == 0 {
				onProgress(indexed, total, skipped)
			}
			continue
		}

		vec, err := Embed(cfg, msg.Text)
		if err != nil {
			continue // skip failures, don't abort
		}

		points = append(points, map[string]any{
			"id":     hashID(msg.ID),
			"vector": vec,
			"payload": map[string]any{
				"id":        msg.ID,
				"text":      msg.Text,
				"platform":  string(msg.Platform),
				"timestamp": msg.Timestamp.Format(time.RFC3339),
			},
		})

		indexed++

		if onProgress != nil && indexed%50 == 0 {
			onProgress(indexed, total, skipped)
		}

		// batch upsert every 100 points
		if len(points) >= 100 {
			if err := upsertPoints(cfg, points); err != nil {
				return indexed, err
			}
			points = points[:0]
		}
	}

	// upsert remaining
	if len(points) > 0 {
		if err := upsertPoints(cfg, points); err != nil {
			return indexed, err
		}
	}

	return indexed, nil
}

// Search returns the most similar messages to the query text.
func Search(cfg RAGConfig, query string, platform string, limit int) ([]corpus.RawMessage, error) {
	vec, err := Embed(cfg, query)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"vector":       vec,
		"limit":        limit,
		"with_payload": true,
	}

	// filter by platform if specified
	if platform != "" {
		body["filter"] = map[string]any{
			"must": []map[string]any{
				{
					"key": "platform",
					"match": map[string]any{
						"value": platform,
					},
				},
			},
		}
	}

	jsonBody, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/collections/%s/points/search", cfg.QdrantURL, collectionName)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result []struct {
			Payload struct {
				ID        string `json:"id"`
				Text      string `json:"text"`
				Platform  string `json:"platform"`
				Timestamp string `json:"timestamp"`
			} `json:"payload"`
			Score float64 `json:"score"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	msgs := make([]corpus.RawMessage, 0, len(result.Result))
	for _, r := range result.Result {
		ts, _ := time.Parse(time.RFC3339, r.Payload.Timestamp)
		msgs = append(msgs, corpus.RawMessage{
			ID:        r.Payload.ID,
			Text:      r.Payload.Text,
			Platform:  corpus.Platform(r.Payload.Platform),
			Timestamp: ts,
			IsFromUser: true,
		})
	}

	return msgs, nil
}

// IsAvailable checks if Qdrant and Ollama are reachable.
func IsAvailable(cfg RAGConfig) bool {
	resp, err := httpClient.Get(cfg.QdrantURL + "/collections")
	if err != nil {
		return false
	}
	resp.Body.Close()

	resp, err = httpClient.Get(cfg.OllamaURL + "/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()

	return true
}

func upsertPoints(cfg RAGConfig, points []map[string]any) error {
	body := map[string]any{"points": points}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/collections/%s/points?wait=true", cfg.QdrantURL, collectionName)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant upsert failed: %d %s", resp.StatusCode, string(b))
	}

	return nil
}

func getExistingIDs(cfg RAGConfig, messages []corpus.RawMessage) (map[string]bool, error) {
	ids := make([]uint64, len(messages))
	for i, m := range messages {
		ids[i] = hashID(m.ID)
	}

	body := map[string]any{
		"ids":          ids,
		"with_payload": false,
		"with_vector":  false,
	}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/collections/%s/points", cfg.QdrantURL, collectionName)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// collection might not exist yet
		return map[string]bool{}, nil
	}

	var result struct {
		Result []struct {
			ID      uint64 `json:"id"`
			Payload struct {
				ID string `json:"id"`
			} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return map[string]bool{}, nil
	}

	existing := make(map[string]bool)
	// map numeric IDs back to string IDs
	idMap := make(map[uint64]string)
	for _, m := range messages {
		idMap[hashID(m.ID)] = m.ID
	}
	for _, r := range result.Result {
		if sid, ok := idMap[r.ID]; ok {
			existing[sid] = true
		}
	}

	return existing, nil
}

// hashID converts a string ID to a uint64 for Qdrant point ID.
// Uses FNV-1a hash.
func hashID(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
