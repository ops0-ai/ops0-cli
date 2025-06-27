package main

// ANSI escape codes for text formatting
const (
	bold      = "\033[1m"
	reset     = "\033[0m"
	blue      = "\033[34m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	underline = "\033[4m"
)


// Claude API configuration
type ClaudeConfig struct {
	APIKey string
	Model  string
	MaxTokens int
}

// Claude API request/response structures
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type Tool struct {
	Name        string
	CheckCmd    string
	InstallCmd  string
	IsInstalled bool
}

type CommandSuggestion struct {
	Tool        string
	Command     string
	DryRunCommand string  // Command to use for dry run
	Description string
	Intent      string
	Confidence  float64
	AIGenerated bool
	HasDryRun   bool     // Whether this command supports dry run
}

type cmdCount struct {
	cmd   string
	count int
}
