package main

import (
	"fmt"
	"os"
	"strings"
)

// Helper to parse AI description for file names and YAML blocks
func parseAnsibleFilesFromAIDescription(desc string) (map[string]string, error) {
	files := make(map[string]string)
	lines := strings.Split(desc, "\n")
	var currentFile string
	var currentContent []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasSuffix(line, "with:") && !strings.Contains(line, "Then ") {
			if currentFile != "" && len(currentContent) > 0 {
				files[currentFile] = strings.Join(currentContent, "\n")
			}
			currentFile = strings.TrimSuffix(line, " with:")
			currentContent = []string{}
			continue
		}
		if currentFile != "" {
			if line == "AI Confidence: 85%" || strings.HasPrefix(line, "Would you like to execute") || strings.HasPrefix(line, "Command:") {
				files[currentFile] = strings.Join(currentContent, "\n")
				currentFile = ""
				currentContent = []string{}
				continue
			}
			currentContent = append(currentContent, lines[i])
		}
	}
	if currentFile != "" && len(currentContent) > 0 {
		files[currentFile] = strings.Join(currentContent, "\n")
	}
	return files, nil
}

func findAnsiblePlaybookAndInventory(files map[string]string) (string, string) {
	playbookFile := ""
	inventoryFile := ""
	for fname := range files {
		if strings.Contains(fname, "playbook") || strings.HasSuffix(fname, ".yml") && playbookFile == "" {
			playbookFile = fname
		}
		if strings.Contains(fname, "inventory") || strings.HasPrefix(fname, "inv") {
			inventoryFile = fname
		}
	}
	if playbookFile == "" {
		playbookFile = "playbook.yml"
	}
	if inventoryFile == "" {
		inventoryFile = "inventory.yml"
	}
	return playbookFile, inventoryFile
}


func generateAnsibleProjectAIWithFilenames(userMsg string) (string, string, string, string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", "", "", "", fmt.Errorf("No ANTHROPIC_API_KEY set")
	}
	prompt := `You are an expert DevOps assistant. Given the following user request, generate:
1. A complete Ansible playbook YAML (for playbook file)
2. A valid Ansible inventory file (for inventory file)

Respond in this format:
---PLAYBOOK FILE---
<playbook filename>
---PLAYBOOK---
<playbook yaml>
---INVENTORY FILE---
<inventory filename>
---INVENTORY---
<inventory content>

User request: ` + userMsg
	claudeConfig := &ClaudeConfig{
		APIKey: apiKey,
		Model:  "claude-3-5-sonnet-20241022",
		MaxTokens: 1024,
	}
	response := callClaude(claudeConfig, prompt, "")
	if response == "" {
		return "", "", "", "", fmt.Errorf("AI did not return a response")
	}
	playbookContent, inventoryContent, playbookFile, inventoryFile := parseAnsibleAIResponseWithFilenames(response)
	if playbookFile == "" {
		playbookFile = "playbook.yml"
	}
	if inventoryFile == "" {
		inventoryFile = "inventory.yml"
	}
	return playbookContent, inventoryContent, playbookFile, inventoryFile, nil
}

func parseAnsibleAIResponseWithFilenames(resp string) (string, string, string, string) {
	playbook := ""
	inventory := ""
	playbookFile := ""
	inventoryFile := ""
	pfStart := strings.Index(resp, "---PLAYBOOK FILE---")
	pStart := strings.Index(resp, "---PLAYBOOK---")
	ifStart := strings.Index(resp, "---INVENTORY FILE---")
	iStart := strings.Index(resp, "---INVENTORY---")
	if pfStart != -1 && pStart != -1 {
		playbookFile = strings.TrimSpace(resp[pfStart+len("---PLAYBOOK FILE---"):pStart])
	}
	if pStart != -1 && ifStart != -1 {
		playbook = strings.TrimSpace(resp[pStart+len("---PLAYBOOK---"):ifStart])
	}
	if ifStart != -1 && iStart != -1 {
		inventoryFile = strings.TrimSpace(resp[ifStart+len("---INVENTORY FILE---"):iStart])
	}
	if iStart != -1 {
		inventory = strings.TrimSpace(resp[iStart+len("---INVENTORY---"):])
	}
	return playbook, inventory, playbookFile, inventoryFile
}

func generateAnsibleProjectTemplate(userMsg string) (string, string, error) {
	// Simple fallback: extract project name, group, host (very basic)
	project := "ansible-project"
	group := "web"
	host := "127.0.0.1"
	if strings.Contains(userMsg, "nginx") {
		group = "nginx"
	}
	if ip := extractIP(userMsg); ip != "" {
		host = ip
	}
	playbook := fmt.Sprintf(`- name: %s
  hosts: %s
  become: yes
  tasks:
    - name: Install nginx
      apt:
        name: nginx
        state: present
      when: ansible_os_family == 'Debian'
    - name: Restart nginx
      service:
        name: nginx
        state: restarted
    - name: Create symlink
      file:
        src: /some/source
        dest: /some/dest
        state: link
`, project, group)
	inventory := fmt.Sprintf(`[%s]
%s
`, group, host)
	return playbook, inventory, nil
}