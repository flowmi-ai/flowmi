package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/flowmi/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var scrapeCmd = &cobra.Command{
	Use:   "scrape <url>",
	Short: "Scrape a web page",
	Long:  `Fetch and extract content from a web page. Returns plain text by default, or markdown with --markdown.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runScrape,
}

func init() {
	scrapeCmd.Flags().BoolP("markdown", "m", false, "Return content as markdown")

	rootCmd.AddCommand(scrapeCmd)
}

func runScrape(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	markdown, _ := cmd.Flags().GetBool("markdown")

	result, err := client.Scrape(cmd.Context(), &api.ScrapeRequest{
		URL:             args[0],
		IncludeMarkdown: markdown,
	})
	if err != nil {
		return fmt.Errorf("scraping: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case "text", "":
		return printScrapeText(cmd, result, markdown)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printScrapeText(cmd *cobra.Command, result *api.ScrapeResponse, markdown bool) error {
	w := cmd.OutOrStdout()
	if title := result.Metadata["title"]; title != "" {
		fmt.Fprintf(w, "Title: %s\n\n", title)
	}
	if markdown && result.Markdown != "" {
		fmt.Fprintln(w, result.Markdown)
	} else {
		fmt.Fprintln(w, result.Text)
	}
	return nil
}
