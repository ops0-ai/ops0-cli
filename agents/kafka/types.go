package main

import (
	"strconv"
	"time"
)

// CommandSuggestion represents a suggested command based on user input
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

// Helper functions that are used across multiple files
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// Helper functions for map operations
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true"
		}
	}
	return false
} 