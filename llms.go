package main

import (
	"fmt"
	"os"
	"time"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Functions for calling specific LLMs Mostly Claude for now.

func callClaude(config *ClaudeConfig, systemPrompt, userMessage string) string {
	request := ClaudeRequest{
		Model:     config.Model,
		MaxTokens: config.MaxTokens,
		System:    systemPrompt,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("⚠️  ops0: Error preparing AI request: %v\n", err)
		return ""
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("⚠️  ops0: Error creating AI request: %v\n", err)
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("⚠️  ops0: Error calling AI service: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("⚠️  ops0: Error reading AI response: %v\n", err)
		return ""
	}

	if resp.StatusCode != 200 {
		fmt.Printf("⚠️  ops0: AI service error (status %d): %s\n", resp.StatusCode, string(body))
		return ""
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		fmt.Printf("⚠️  ops0: Error parsing AI response: %v\n", err)
		return ""
	}

	if len(claudeResp.Content) > 0 {
		return claudeResp.Content[0].Text
	}

	return ""
}

func getClaudeConfigIfAvailable() *ClaudeConfig {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil
	}
	model := os.Getenv("OPS0_AI_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &ClaudeConfig{
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: 1024,
	}
}
