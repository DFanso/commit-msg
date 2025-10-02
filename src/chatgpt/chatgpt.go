package chatgpt

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/dfanso/commit-msg/src/types"
)

func GenerateCommitMessage(config *types.Config, changes string, apiKey string) (string, error) {
	
	client := openai.NewClient(option.WithAPIKey(apiKey))

	prompt := fmt.Sprintf("%s\n\n%s", types.CommitPrompt, changes)

	resp, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model:       openai.ChatModelGPT4o,
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI error: %w", err)
	}

	// Extract and return the commit message
	commitMsg := resp.Choices[0].Message.Content
	return commitMsg, nil
}