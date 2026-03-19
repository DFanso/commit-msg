package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	setTemplate    bool
	getTemplate    bool
	deleteTemplate bool
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage commit message templates",
	Long: `Manage the commit message templates for LLM-generated commits.

You can set, view, or delete custom templates that will be used by the LLM
to generate commit messages in your preferred format.

Examples:
	commit llm template --set			# set a new custom template
	commit llm template --get			# view the current template
	commit llm template --delete		# delete the custom template`,
	Run: func(cmd *cobra.Command, args []string) {
		if setTemplate {
			handleSetTemplate()
		} else if getTemplate {
			handleGetTemplate()
		} else if deleteTemplate {
			handleDeleteTemplate()
		} else {
			cmd.Help()
		}
	},
}

func init() {
	llmCmd.AddCommand(templateCmd)
	templateCmd.Flags().BoolVar(&setTemplate, "set", false, "Set a custom commit message template")
	templateCmd.Flags().BoolVar(&getTemplate, "get", false, "Get the current custom template")
	templateCmd.Flags().BoolVar(&deleteTemplate, "delete", false, "Delete the custom template")
}

func handleSetTemplate() {
	fmt.Println("Custom Commit Message Template Setup")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Enter your custom commit message template.")
	fmt.Println("You can use placeholders that the LLM will fill in:")
	fmt.Println("  - Use natural language to describe your desired format")
	fmt.Println("  - Example: 'Type: Brief description\\n\\nDetailed explanation\\n\\nRelated: #issue'")
	fmt.Println()
	fmt.Println("Enter your template (press Ctrl+D or Ctrl+Z when done):")
	fmt.Println("----------------------------------------")

	// Read multiline input
	scanner := bufio.NewScanner(os.Stdin)
	var templateLines []string

	for scanner.Scan() {
		templateLines = append(templateLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("x Error reading template: %v\n", err)
		return
	}

	template := strings.Join(templateLines, "\n")
	template = strings.TrimSpace(template)

	if template == "" {
		fmt.Println("x Template cannot be empty.")
		return
	}

	// save template to store
	err := Store.SaveTemplate(template)
	if err != nil {
		fmt.Printf("x Error saving template: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Custom template saved successfully!")
	fmt.Println()
	fmt.Println("Your template:")
	fmt.Println("----------------------------------------")
	fmt.Println(template)
	fmt.Println("----------------------------------------")
	fmt.Println()
	fmt.Println("💡 This template will now be used for all commit message generation.")
	fmt.Println("   Run 'commit llm template --get' to view it anytime.")
	fmt.Println("   Run 'commit llm template --delete' to remove it.")

}

func handleGetTemplate() {
	template, err := Store.GetTemplate()
	if err != nil {
		if err.Error() == "no custom template set" {
			fmt.Println("No custom template is currently set.")
			fmt.Println()
			fmt.Println("To set a custom template, run: commit llm template --set")
			fmt.Println("   The default template will be used for commit message generation.")
			return
		}
		fmt.Printf("Error retrieving template: %v\n", err)
		return
	}

	fmt.Println("📋 Current Custom Template")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println(template)
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("💡 To update this template, run: commit llm template --set")
	fmt.Println("   To delete this template, run: commit llm template --delete")
}

func handleDeleteTemplate() {
	_, err := Store.GetTemplate()
	if err != nil {
		if err.Error() == "no custom template set" {
			fmt.Println("No custom template is currently set.")
			fmt.Println()
			fmt.Println("To set a custom template, run: commit llm template --set")
			fmt.Println("   The default template will be used for commit message generation.")
			return
		}
		fmt.Printf("Error retrieving template: %v\n", err)
		return
	}

	// Confirm deletion
	fmt.Println("Are you sure you want to delete the custom template?")
	fmt.Println("   The default template will be used after deletion.")
	fmt.Print("   Type 'yes' to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("❌ Error reading confirmation: %v\n", err)
		return
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" {
		fmt.Println("❌ Template deletion cancelled.")
		return
	}

	// Delete the template
	err = Store.DeleteTemplate()
	if err != nil {
		fmt.Printf("❌ Error deleting template: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Custom template deleted successfully!")
	fmt.Println("   The default template will now be used for commit message generation.")
}
