package types

import (
	"fmt"
	"strings"
)

// CommitPrompt is the base instruction template sent to LLM providers before
// appending repository changes and optional style guidance.
var CommitPrompt = `I need a concise git commit message based on the following changes from my Git repository.
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
`

// BuildCommitPrompt constructs the prompt that will be sent to the LLM, applying
// any optional tone/style instructions before appending the repository changes.
func BuildCommitPrompt(changes string, opts *GenerationOptions) string {
	return BuildCommitPromptWithTemplate(changes, opts, "")
}

func BuildCommitPromptWithTemplate(changes string, opts *GenerationOptions, customTemplate string) string {
	var builder strings.Builder

	// use custom template if provided
	basePrompt := CommitPrompt
	if strings.TrimSpace(customTemplate) != "" {
		basePrompt = strings.TrimSpace(customTemplate) + "\n\nHere are the changes:\n"
	}

	builder.WriteString(basePrompt)

	if opts != nil {
		if opts.Attempt > 1 {
			builder.WriteString("\n\nRegeneration context:\n")
			builder.WriteString(fmt.Sprintf("- This is attempt #%d.\n", opts.Attempt))
			builder.WriteString("- Provide a commit message that is meaningfully different from earlier attempts.\n")
		}

		if strings.TrimSpace(opts.StyleInstruction) != "" {
			builder.WriteString("\n\nAdditional instructions:\n")
			builder.WriteString(strings.TrimSpace(opts.StyleInstruction))
		}
	}

	builder.WriteString("\n\n")
	builder.WriteString(changes)

	return builder.String()
}
