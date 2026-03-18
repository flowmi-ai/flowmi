package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var scrapeCmd = &cobra.Command{
	Use:   "scrape <url>",
	Short: "Scrape a web page",
	Long:  `Fetch and extract content from a web page as markdown.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runScrape,
}

func init() {
	rootCmd.AddCommand(scrapeCmd)
}

func runScrape(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}
	client.HTTPClient.SetTimeout(60 * time.Second)

	result, err := client.Scrape(cmd.Context(), &api.ScrapeRequest{
		URL:             args[0],
		IncludeMarkdown: true,
	})
	if err != nil {
		return fmt.Errorf("scraping: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	return printScrapeText(cmd, result)
}

func printScrapeText(cmd *cobra.Command, result *api.ScrapeResponse) error {
	fmt.Fprintln(cmd.OutOrStdout(), result.Markdown)
	return nil
}
