package chatgpt

import (
	"context"
	"fmt"

	"github.com/dfanso/commit-msg/cmd/cli/store"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/dfanso/commit-msg/pkg/types"
)

var storeMethods *store.StoreMethods

const (
	chatgptModel = openai.ChatModelGPT4o
)

// GenerateCommitMessage calls OpenAI's chat completions API to turn the provided
// repository changes into a polished git commit message.
func GenerateCommitMessage(config *types.Config, changes string, apiKey string, opts *types.GenerationOptions) (string, error) {

	client := openai.NewClient(option.WithAPIKey(apiKey))

	// getting the custom template
	customTemplate, _ := storeMethods.GetTemplate()
	prompt := types.BuildCommitPromptWithTemplate(changes, opts, customTemplate)

	resp, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model: chatgptModel,
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI error: %w", err)
	}

	// Extract and return the commit message
	commitMsg := resp.Choices[0].Message.Content
	return commitMsg, nil
}
