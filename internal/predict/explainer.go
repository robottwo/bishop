package predict

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/robottwo/bishop/internal/docs"
	"github.com/robottwo/bishop/internal/environment"
	"github.com/robottwo/bishop/internal/utils"
	"github.com/robottwo/bishop/pkg/gline"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

type LLMExplainer struct {
	runner      *interp.Runner
	llmClient   *openai.Client
	contextText string
	logger      *zap.Logger
	modelId     string
	temperature *float64
}

func NewLLMExplainer(
	runner *interp.Runner,
	logger *zap.Logger,
) *LLMExplainer {
	llmClient, modelConfig := utils.GetLLMClient(runner, utils.FastModel)
	return &LLMExplainer{
		runner:      runner,
		llmClient:   llmClient,
		contextText: "",
		logger:      logger,
		modelId:     modelConfig.ModelId,
		temperature: modelConfig.Temperature,
	}
}

func (p *LLMExplainer) UpdateContext(context *map[string]string) {
	contextTypes := environment.GetContextTypesForExplanation(p.runner, p.logger)
	p.contextText = utils.ComposeContextText(context, contextTypes, p.logger)
}

func (e *LLMExplainer) Explain(input string) (*gline.Explanation, error) {
	if input == "" {
		return nil, nil
	}

	// Get documentation
	docs, err := docs.GetRelevantDocumentation(context.TODO(), e.runner, e.logger, input)
	if err != nil {
		e.logger.Warn("failed to get documentation", zap.Error(err))
		// Proceed without docs
	}

	schema, err := EXPLAINED_COMMAND_SCHEMA.MarshalJSON()
	if err != nil {
		return nil, err
	}

	systemMessage := fmt.Sprintf(`You are Bishop, an intelligent shell program.
You will be given a bash command entered by me, enclosed in <command> tags.

# Instructions
* Check the command for:
  - Syntax errors
  - Incorrect parameters (refer to provided documentation if available)
  - Dangerous commands (e.g. rm -rf /, recursive chmod, etc.)
* If any errors or dangers are found, provide a concise error message in the 'error' field.
* Give a concise explanation of what the command will do for me in the 'explanation' field.
* If any uncommon arguments are present in the command, 
  format your explanation in markdown and explain arguments in a bullet point list

# Documentation Reference
%s

# Latest Context
%s

# Response JSON Schema
%s`,
		docs,
		e.contextText,
		string(schema),
	)

	userMessage := fmt.Sprintf(
		`<command>%s</command>`,
		input,
	)

	e.logger.Debug(
		"explaining prediction using LLM",
		zap.String("system", systemMessage),
		zap.String("user", userMessage),
	)

	request := openai.ChatCompletionRequest{
		Model: e.modelId,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}
	if e.temperature != nil {
		request.Temperature = float32(*e.temperature)
	}

	chatCompletion, err := e.llmClient.CreateChatCompletion(context.TODO(), request)

	if err != nil {
		return nil, err
	}

	explanation := explainedCommand{}
	_ = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &explanation)

	e.logger.Debug(
		"LLM explanation response",
		zap.Any("response", explanation),
	)

	return &gline.Explanation{
		Text:  explanation.Explanation,
		Error: explanation.Error,
	}, nil
}
