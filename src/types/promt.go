package types

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
