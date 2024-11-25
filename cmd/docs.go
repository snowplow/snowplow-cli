/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"github.com/spf13/cobra/doc"
)

var docsCommand = &cobra.Command{
	Use:    "generate-docs [output-dir]",
	Short:  "Generate markdown documentation for snowplow-cli",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputDir := args[0]

		// Clean the output directory if it exists
		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			if err := os.RemoveAll(outputDir); err != nil {
				slog.Error("Failed to clean output directory", "error", err)
				os.Exit(1)
			}
		}

		// Create fresh output directory
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			slog.Error("Failed to create output directory", "error", err)
			os.Exit(1)
		}

		// Generate markdown docs
		if err := doc.GenMarkdownTree(RootCmd, outputDir); err != nil {
			slog.Error("Failed to generate markdown documentation", "error", err)
			os.Exit(1)
		}

		// Collect all documentation content
		var allContent []struct {
			title   string
			content string
		}

		err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") &&
				!strings.Contains(info.Name(), "completion") &&
				info.Name() != "snowplow-cli.md" {

				content, err := processDocFile(path)
				if err != nil {
					return err
				}

				title := strings.TrimSuffix(info.Name(), ".md")
				title = strings.TrimPrefix(title, "snowplow-cli_")
				title = strings.ReplaceAll(title, "_", " ")

				allContent = append(allContent, struct {
					title   string
					content string
				}{title, content})

				// Remove the processed file
				os.Remove(path)
			}

			// Remove autocompletion docs and root command doc
			if !info.IsDir() && (strings.Contains(info.Name(), "completion") || info.Name() == "snowplow-cli.md") {
				os.Remove(path)
			}

			return nil
		})

		if err != nil {
			slog.Error("Failed to process documentation files", "error", err)
			os.Exit(1)
		}

		// Create the combined documentation file
		var combinedContent strings.Builder

		// Add front matter
		combinedContent.WriteString("---\n")
		combinedContent.WriteString("title: Command Reference\n")
		combinedContent.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format("2006-01-02")))
		combinedContent.WriteString("sidebar_label: Command Reference\n")
		combinedContent.WriteString("sidebar_position: 1\n")
		combinedContent.WriteString("---\n\n")

		combinedContent.WriteString("This page contains the complete reference for the Snowplow CLI commands.\n\n")

		// Add each command's content with appropriate heading
		for _, doc := range allContent {
			caser := cases.Title(language.English)
			combinedContent.WriteString(fmt.Sprintf("## %s\n\n", caser.String(doc.title)))
			combinedContent.WriteString(doc.content)
			combinedContent.WriteString("\n\n")
		}

		// Write the combined file
		outputFile := filepath.Join(outputDir, "index.md")
		if err := os.WriteFile(outputFile, []byte(combinedContent.String()), 0644); err != nil {
			slog.Error("Failed to write combined documentation", "error", err)
			os.Exit(1)
		}

		if err != nil {
			slog.Error("Failed to convert documentation to Docusaurus format", "error", err)
			os.Exit(1)
		}

		slog.Info("Documentation generated successfully", "path", filepath.Clean(outputDir))
	},
}

func processDocFile(filepath string) (string, error) {
	// Read the file
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")

	// Extract title from the first heading
	title := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			title = strings.TrimPrefix(line, "## ")
			break
		}
	}

	var newContent strings.Builder

	// Process content
	inCodeBlock := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## "+title) {
			continue
		}

		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if strings.HasPrefix(line, "```markdown") {
				// Skip markdown code block markers
				continue
			}
		}

		// Add line to new content with escaped angle brackets
		escapedLine := strings.ReplaceAll(strings.ReplaceAll(line, "<", "\\<"), ">", "\\>")

		// Remove link to the autocomplete command
		if !strings.HasPrefix(escapedLine, "### SEE ALSO") {
			newContent.WriteString(escapedLine + "\n")
		} else {
			break
		}
	}

	// Remove the auto-generated notice
	final := strings.Replace(newContent.String(),
		"###### Auto generated by spf13/cobra on "+time.Now().Format("2-Jan-2006")+"\n",
		"", -1)

	return final, nil
}

func init() {
	RootCmd.AddCommand(docsCommand)
}
