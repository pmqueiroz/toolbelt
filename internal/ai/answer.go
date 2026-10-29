package ai

import (
	"encoding/json"
	"strings"
)

type BlockType string

const (
	Text    BlockType = "text"
	Command BlockType = "command"
	Code    BlockType = "code"
	List    BlockType = "list"
	Warning BlockType = "warning"
)

var blockTypes = []BlockType{Text, Command, Code, List, Warning}

type Block struct {
	Type     BlockType `json:"type"`
	Text     string    `json:"text"`
	Code     string    `json:"code"`
	Language string    `json:"language"`
	Items    []string  `json:"items"`
}

type Answer struct {
	Blocks []Block `json:"blocks"`
}

func (a Answer) Commands() []string {
	var commands []string
	for _, b := range a.Blocks {
		if b.Type == Command && b.Code != "" {
			commands = append(commands, b.Code)
		}
	}
	return commands
}

func Schema() map[string]any {
	types := make([]string, len(blockTypes))
	for i, t := range blockTypes {
		types[i] = string(t)
	}
	block := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":     map[string]any{"type": "string", "enum": types},
			"text":     map[string]any{"type": "string"},
			"code":     map[string]any{"type": "string"},
			"language": map[string]any{"type": "string"},
			"items":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required":             []string{"type", "text", "code", "language", "items"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"blocks": map[string]any{"type": "array", "items": block},
		},
		"required":             []string{"blocks"},
		"additionalProperties": false,
	}
}

func Parse(raw string) Answer {
	raw = strings.TrimSpace(raw)
	var answer Answer
	if body, ok := jsonObject(raw); ok && json.Unmarshal([]byte(body), &answer) == nil {
		if blocks := validBlocks(answer.Blocks); len(blocks) > 0 {
			return Answer{Blocks: blocks}
		}
	}
	if raw == "" {
		return Answer{}
	}
	return Answer{Blocks: []Block{{Type: Text, Text: raw}}}
}

func jsonObject(raw string) (string, bool) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return "", false
	}
	return raw[start : end+1], true
}

func validBlocks(blocks []Block) []Block {
	valid := make([]Block, 0, len(blocks))
	for _, b := range blocks {
		b.Text = strings.TrimSpace(b.Text)
		b.Code = strings.TrimSpace(b.Code)
		b.Items = nonEmpty(b.Items)
		switch b.Type {
		case Text, Warning:
			if b.Text == "" {
				continue
			}
		case Command, Code:
			if b.Code == "" {
				continue
			}
		case List:
			if len(b.Items) == 0 {
				continue
			}
		default:
			continue
		}
		valid = append(valid, b)
	}
	return valid
}

func nonEmpty(items []string) []string {
	var kept []string
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			kept = append(kept, item)
		}
	}
	return kept
}
