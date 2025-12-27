package docs

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/robottwo/bishop/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

type ManPageProvider struct {
	cache sync.Map // map[string]string
}

var globalProvider = &ManPageProvider{}

// GetRelevantDocumentation fetches and summarizes documentation for commands in the input.
func GetRelevantDocumentation(ctx context.Context, runner *interp.Runner, logger *zap.Logger, input string) (string, error) {
	commands := extractCommands(input)
	if len(commands) == 0 {
		return "", nil
	}

	var docsBuilder strings.Builder
	for _, cmd := range commands {
		doc, err := globalProvider.getDoc(ctx, runner, logger, cmd)
		if err != nil {
			logger.Debug("failed to get doc for command", zap.String("command", cmd), zap.Error(err))
			continue
		}
		if doc != "" {
			docsBuilder.WriteString(fmt.Sprintf("\n--- Documentation for %s ---\n%s\n", cmd, doc))
		}
	}

	return docsBuilder.String(), nil
}

func (p *ManPageProvider) getDoc(ctx context.Context, runner *interp.Runner, logger *zap.Logger, command string) (string, error) {
	if val, ok := p.cache.Load(command); ok {
		return val.(string), nil
	}

	// Try fetching
	content, err := fetchManPage(command)
	if err != nil {
		// Log but don't fail hard, just return empty
		logger.Debug("man page not found", zap.String("command", command), zap.Error(err))
		p.cache.Store(command, "")
		return "", nil
	}

	if content == "" {
		p.cache.Store(command, "")
		return "", nil
	}

	// Summarize if too long
	// Use heuristic limit (e.g. 10000 chars)
	if len(content) > 10000 {
		summarized, err := summarizeDoc(ctx, runner, logger, command, content)
		if err == nil && summarized != "" {
			content = summarized
		} else {
			// Fallback: simple truncation
			logger.Warn("summarization failed, using truncation", zap.Error(err))
			if len(content) > 10000 {
				content = content[:10000] + "\n... (truncated)"
			}
		}
	}

	p.cache.Store(command, content)
	return content, nil
}

func fetchManPage(command string) (string, error) {
	// Security check: ensure command only contains safe characters
	// Allow alphanumeric, underscore, dash, dot
	if strings.ContainsAny(command, ";|&`$()<>\\\"'") || strings.Contains(command, "/") {
		return "", fmt.Errorf("invalid characters in command name")
	}

	// Check if man exists
	_, err := exec.LookPath("man")
	if err != nil {
		return "", fmt.Errorf("man command not found")
	}

	// Try man -P cat <command>
	cmd := exec.Command("man", "-P", "cat", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	// Don't capture stderr to avoid noise, or capture it to debug?
	// Man usually writes errors to stderr
	if err := cmd.Run(); err == nil {
		return out.String(), nil
	}

	// Try --help as fallback?
	// Risk: some commands execute on --help.
	// Safest is to avoid execution.

	return "", fmt.Errorf("no documentation found")
}

func summarizeDoc(ctx context.Context, runner *interp.Runner, logger *zap.Logger, command string, content string) (string, error) {
	client, modelConfig := utils.GetLLMClient(runner, utils.FastModel)

	// Truncate content sent to summarizer to avoid blowing up context window of the summarizer itself
	// Assuming fast model has at least 16k context ~ 50k chars.
	// We'll limit to 40k chars to be safe + prompt.
	inputLen := len(content)
	if inputLen > 40000 {
		content = content[:40000] + "\n...(input truncated)"
	}

	prompt := fmt.Sprintf(`You are a documentation assistant.
Refine the following man page for the command '%s' into a concise reference.
Keep the SYNOPSIS, and relevant OPTIONS or DESCRIPTION that are most commonly used.
Remove verbose copyright info, authors, or obscure details.
Limit output to around 2000 characters if possible.

Man Page Content:
%s`, command, content)

	req := openai.ChatCompletionRequest{
		Model: modelConfig.ModelId,
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
	}
	if modelConfig.Temperature != nil {
		req.Temperature = float32(*modelConfig.Temperature)
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response from LLM")
}

func extractCommands(input string) []string {
	var commands []string
	p := syntax.NewParser()
	file, err := p.Parse(strings.NewReader(input), "")
	if err != nil {
		return nil
	}

	syntax.Walk(file, func(node syntax.Node) bool {
		if cmd, ok := node.(*syntax.CallExpr); ok {
			if len(cmd.Args) > 0 {
				// The first argument is the command
				if len(cmd.Args[0].Parts) == 1 {
					if lit, ok := cmd.Args[0].Parts[0].(*syntax.Lit); ok {
						commands = append(commands, lit.Value)
					}
				}
			}
		}
		return true
	})

	// Deduplicate
	seen := make(map[string]bool)
	var result []string
	for _, cmd := range commands {
		if !seen[cmd] {
			seen[cmd] = true
			result = append(result, cmd)
		}
	}
	return result
}
