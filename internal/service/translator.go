package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Translator struct {
	apiKey  string
	model   string
	client  *http.Client
}

func NewTranslator(apiKey, model string) *Translator {
	return &Translator{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

const systemPrompt = `You are a professional technical book translator.
Translate the following English text into natural, fluent Indonesian.
Rules:
1. Preserve ALL code blocks exactly as-is — do NOT translate any code.
2. Keep software engineering technical terms in English (e.g., function, variable, method, API, framework, struct, interface, package, goroutine).
3. Output clean HTML: wrap paragraphs in <p>, code in <pre><code>, headings in <h2> or <h3>.
4. Output ONLY the translated HTML — no explanations, no markdown fences, no extra commentary.`

const deepSeekBaseURL = "https://api.deepseek.com/v1"

func (t *Translator) Translate(ctx context.Context, text string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: text},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deepSeekBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("api error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from api")
	}

	return result.Choices[0].Message.Content, nil
}
