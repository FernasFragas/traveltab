package guides

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// defaultOllamaURL is where a local Ollama server listens by default.
	defaultOllamaURL = "http://localhost:11434"
	// defaultModel is the local model used unless the caller selects another.
	defaultModel = "mistral:7b"
	// ollamaTimeout is generous: a 7B model on CPU can take a while for a ~120-word answer.
	ollamaTimeout = 2 * time.Minute
)

// introPrompt is the instruction sent with the source text. It asks for a short, general intro
// and is explicit that the model must not invent places beyond what the source mentions.
const introPrompt = `You are writing a short introduction to a city for a trip-planning app.

Using ONLY the following Wikivoyage text about %s, write a general introduction of no more than
120 words: why someone would want to visit, what the different areas are like, and how to get
around. Keep strictly to 120 words or fewer. Write it as a single paragraph of plain prose, with
no headings, no lists, and no history lesson — focus on what a visitor experiences today. Avoid
naming specific landmarks, museums, or monuments; describe them generally instead (e.g. "grand
museums" rather than naming one) unless a place is essential to describing an area. Do not
mention any place, neighborhood, landmark, or district that is not named, using the same words,
in the text below. Do not invent details.

Wikivoyage text:
%s`

// OllamaClient asks a local Ollama server to generate text with a given model.
type OllamaClient struct {
	client  *http.Client
	baseURL string
	model   string
}

// NewOllamaClient returns a client for a local Ollama server. A nil client means a default one
// with a generous timeout; an empty baseURL or model falls back to the local default and
// mistral:7b respectively.
func NewOllamaClient(client *http.Client, baseURL, model string) *OllamaClient {
	if client == nil {
		client = &http.Client{Timeout: ollamaTimeout}
	}

	if baseURL == "" {
		baseURL = defaultOllamaURL
	}

	if model == "" {
		model = defaultModel
	}

	return &OllamaClient{client: client, baseURL: baseURL, model: model}
}

// generateRequest is the body /api/generate expects.
type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// generateResponse is the body /api/generate returns when Stream is false.
type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// GenerateIntro asks the model for a ~120-word intro to cityName, grounded only in sourceText.
func (o *OllamaClient) GenerateIntro(ctx context.Context, cityName, sourceText string) (string, error) {
	prompt := fmt.Sprintf(introPrompt, cityName, sourceText)

	payload, err := json.Marshal(generateRequest{Model: o.model, Prompt: prompt, Stream: false})
	if err != nil {
		return "", fmt.Errorf("encoding ollama request for %s: %w", cityName, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("building ollama request for %s: %w", cityName, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling ollama for %s: %w", cityName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading ollama response for %s: %w", cityName, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama for %s: status %d: %s", cityName, resp.StatusCode, body)
	}

	var parsed generateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decoding ollama response for %s: %w", cityName, err)
	}

	intro := strings.TrimSpace(parsed.Response)
	if intro == "" {
		return "", fmt.Errorf("ollama returned an empty intro for %s", cityName)
	}

	return intro, nil
}
