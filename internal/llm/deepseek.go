package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type ChatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type DeepseekClient struct {
	Model   string `json:"model"`
	BaseURL string `json:"baseURL"`
	APIKey  string `json:"apikey"`
}

func NewDeepseekClient(model string, baseURL string, apiKey string) (*DeepseekClient, error) {
	return &DeepseekClient{
		Model:   model,
		BaseURL: baseURL,
		APIKey:  apiKey,
	}, nil
}

func (dc *DeepseekClient) Call(prompt string) (string, error) {
	return dc.CallStream(prompt, nil)
}

func (dc *DeepseekClient) CallStream(prompt string, onDelta func(string) error) (string, error) {
	messages := []Message{
		{Role: "user", Content: prompt},
	}
	return dc.ChatStream(messages, onDelta)
}

// ChatStream sends an arbitrary set of messages to Deepseek while keeping the stream interface.
func (dc *DeepseekClient) ChatStream(messages []Message, onDelta func(string) error) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("messages cannot be empty")
	}

	payload := ChatRequest{
		Model:       dc.Model,
		Messages:    messages,
		Temperature: 0.7,
		Stream:      true,
	}
	return dc.sendChatRequest(payload, onDelta)
}

func (dc *DeepseekClient) sendChatRequest(payload ChatRequest, onDelta func(string) error) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, dc.BaseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build http request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+dc.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if payload.Stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get resp: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("deepseek error: status %d body %s", resp.StatusCode, snippet)
	}

	if payload.Stream {
		text, err := consumeStream(resp.Body, onDelta)
		if err != nil {
			return "", err
		}
		return text, nil
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("deepseek response missing choices")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func consumeStream(body io.Reader, onDelta func(string) error) (string, error) {
	reader := bufio.NewReader(body)
	var builder strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("read stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || line == ":" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return "", fmt.Errorf("decode chunk: %w", err)
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			builder.WriteString(choice.Delta.Content)
			if onDelta != nil {
				if err := onDelta(choice.Delta.Content); err != nil {
					return "", err
				}
			}
		}
	}

	if builder.Len() == 0 {
		return "", fmt.Errorf("empty response stream")
	}
	return builder.String(), nil
}
