package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/dfanso/commit-msg/src/types"
)

func GenerateCommitMessage(config *types.Config, changes string, apiKey string) (string, error) {
	// Prepare request to Gemini API
	prompt := fmt.Sprintf(`
I need a concise git commit message based on the following changes from my Git repository.
Please generate a commit message that:
1. Starts with a verb in the present tense (e.g., "Add", "Fix", "Update", "Feat", "Refactor", etc.)
2. Is clear and descriptive
3. Focuses on the "what" and "why" of the changes
4. Is no longer than 50-72 characters for the first line
5. Can include a more detailed description after a blank line if needed
6. Dont say any other stuff only include the commit msg

here is a sample commit msgs:
'Feat: Add Gemini LLM support for commit messages

- Adds Gemini as an alternative LLM for generating commit
- messages, configurable via the COMMIT_LLM environment
- variable. Also updates dependencies and README.'
Here are the changes:

%s
`, changes)

	// Create context and client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a GenerativeModel with appropriate settings
	model := client.GenerativeModel("gemini-2.0-flash")
	model.SetTemperature(0.2) // Lower temperature for more focused responses

	// Generate content using the prompt
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	// Check if we got a valid response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	// Extract the commit message from the response
	commitMsg := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return commitMsg, nil
}
