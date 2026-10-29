package ai

import (
	"fmt"
	"strings"
)

const MaxContextBytes = 16 << 10

type Question struct {
	Text    string
	Context string
	OS      string
	Shell   string
}

const instructions = `You answer questions asked from a terminal. Answer with the shortest correct answer: one sentence or one command when that is enough, never an essay. No greetings, no restating the question, no follow-up offers.

Reply only with JSON matching this shape: {"blocks":[{"type":"...","text":"","code":"","language":"","items":[]}]}. Every block has all five fields; leave the ones a type does not use empty.

Block types:
- text: a short plain sentence in "text". No markdown.
- command: one shell command the user can run, in "code"; an optional few-word note in "text". Use it whenever the question asks how to do something in a terminal.
- code: a short snippet in "code", its language in "language".
- list: a few short items in "items", only when the answer is a real list.
- warning: one short caution in "text", only when a command is destructive or risky.

Prefer one block. Use at most three. Commands must fit the user's OS and shell.`

func System(q Question) string {
	return instructions + fmt.Sprintf("\n\nUser OS: %s. Shell: %s.", q.OS, orUnknown(q.Shell))
}

func User(q Question) string {
	text := strings.TrimSpace(q.Text)
	context := strings.TrimSpace(q.Context)
	if context == "" {
		return text
	}
	if len(context) > MaxContextBytes {
		context = context[len(context)-MaxContextBytes:]
	}
	return text + "\n\nInput piped to the command:\n" + context
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
