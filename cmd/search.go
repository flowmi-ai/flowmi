package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/flowmi/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the web",
	Long:  `Search the web, images, or news. Defaults to web search when no subcommand is given.`,
	RunE:  runSearchWeb,
	Args:  cobra.ArbitraryArgs,
}

var searchWebCmd = &cobra.Command{
	Use:   "web <query>",
	Short: "Search the web",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchWeb,
}

var searchImagesCmd = &cobra.Command{
	Use:   "images <query>",
	Short: "Search for images",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchImages,
}

var searchNewsCmd = &cobra.Command{
	Use:   "news <query>",
	Short: "Search for news",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchNews,
}

func init() {
	searchCmd.PersistentFlags().IntP("limit", "L", 10, "Number of results")
	searchCmd.PersistentFlags().IntP("page", "p", 1, "Page number")
	searchCmd.PersistentFlags().String("country", "", "Country code (e.g. us, cn)")
	searchCmd.PersistentFlags().String("language", "", "Language code (e.g. en, zh-cn)")

	searchNewsCmd.Flags().StringP("time", "t", "", "Time range: day, week, month")
	searchImagesCmd.Flags().StringP("size", "s", "", "Image size: large, medium, icon")

	searchCmd.AddCommand(searchWebCmd)
	searchCmd.AddCommand(searchImagesCmd)
	searchCmd.AddCommand(searchNewsCmd)

	rootCmd.AddCommand(searchCmd)
}

func runSearchWeb(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("query is required")
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	query := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	country, _ := cmd.Flags().GetString("country")
	language, _ := cmd.Flags().GetString("language")

	result, err := client.WebSearch(cmd.Context(), &api.WebSearchRequest{
		Q:    query,
		GL:   country,
		HL:   language,
		Num:  limit,
		Page: page,
	})
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case "table":
		return printWebSearchTable(cmd, query, result)
	case "text", "":
		return printWebSearchText(cmd, query, result)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printWebSearchText(cmd *cobra.Command, query string, result *api.WebSearchResponse) error {
	w := cmd.OutOrStdout()
	if len(result.Organic) == 0 {
		fmt.Fprintf(w, "No results found for %q\n", query)
		return nil
	}
	fmt.Fprintf(w, "Showing %d results for %q\n\n", len(result.Organic), query)
	for i, r := range result.Organic {
		fmt.Fprintf(w, "  %2d  %s\n", i+1, r.Title)
		fmt.Fprintf(w, "      %s\n", r.Link)
		if r.Snippet != "" {
			fmt.Fprintf(w, "      %s\n", r.Snippet)
		}
		fmt.Fprintln(w)
	}
	return nil
}

func printWebSearchTable(cmd *cobra.Command, _ string, result *api.WebSearchResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTITLE\tURL")
	for i, r := range result.Organic {
		fmt.Fprintf(w, "%2d\t%s\t%s\n", i+1, truncate(r.Title, 50), r.Link)
	}
	return w.Flush()
}

func runSearchImages(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	query := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	country, _ := cmd.Flags().GetString("country")
	language, _ := cmd.Flags().GetString("language")
	size, _ := cmd.Flags().GetString("size")

	var tbs string
	switch size {
	case "large":
		tbs = "isz:l"
	case "medium":
		tbs = "isz:m"
	case "icon":
		tbs = "isz:i"
	case "":
		// no filter
	default:
		return fmt.Errorf("invalid size %q: must be large, medium, or icon", size)
	}

	result, err := client.ImageSearch(cmd.Context(), &api.ImageSearchRequest{
		Q:    query,
		GL:   country,
		HL:   language,
		Num:  limit,
		Page: page,
		TBS:  tbs,
	})
	if err != nil {
		return fmt.Errorf("searching images: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case "table":
		return printImageSearchTable(cmd, query, result)
	case "text", "":
		return printImageSearchText(cmd, query, result)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printImageSearchText(cmd *cobra.Command, query string, result *api.ImageSearchResponse) error {
	w := cmd.OutOrStdout()
	if len(result.Images) == 0 {
		fmt.Fprintf(w, "No image results found for %q\n", query)
		return nil
	}
	fmt.Fprintf(w, "Showing %d image results for %q\n\n", len(result.Images), query)
	for i, r := range result.Images {
		fmt.Fprintf(w, "  %2d  %s\n", i+1, r.Title)
		fmt.Fprintf(w, "      %s (%d\u00d7%d)\n", r.ImageURL, r.ImageWidth, r.ImageHeight)
		fmt.Fprintf(w, "      Source: %s\n", r.Domain)
		fmt.Fprintln(w)
	}
	return nil
}

func printImageSearchTable(cmd *cobra.Command, _ string, result *api.ImageSearchResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTITLE\tSIZE\tSOURCE")
	for i, r := range result.Images {
		fmt.Fprintf(w, "%2d\t%s\t%d\u00d7%d\t%s\n", i+1, truncate(r.Title, 40), r.ImageWidth, r.ImageHeight, r.Domain)
	}
	return w.Flush()
}

func runSearchNews(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	query := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	country, _ := cmd.Flags().GetString("country")
	language, _ := cmd.Flags().GetString("language")
	timeRange, _ := cmd.Flags().GetString("time")

	var tbs string
	switch timeRange {
	case "day":
		tbs = "qdr:d"
	case "week":
		tbs = "qdr:w"
	case "month":
		tbs = "qdr:m"
	case "":
		// no filter
	default:
		return fmt.Errorf("invalid time range %q: must be day, week, or month", timeRange)
	}

	result, err := client.NewsSearch(cmd.Context(), &api.NewsSearchRequest{
		Q:    query,
		GL:   country,
		HL:   language,
		Num:  limit,
		Page: page,
		TBS:  tbs,
	})
	if err != nil {
		return fmt.Errorf("searching news: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case "table":
		return printNewsSearchTable(cmd, query, result)
	case "text", "":
		return printNewsSearchText(cmd, query, result)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printNewsSearchText(cmd *cobra.Command, query string, result *api.NewsSearchResponse) error {
	w := cmd.OutOrStdout()
	if len(result.News) == 0 {
		fmt.Fprintf(w, "No news results found for %q\n", query)
		return nil
	}
	fmt.Fprintf(w, "Showing %d news results for %q\n\n", len(result.News), query)
	for i, r := range result.News {
		fmt.Fprintf(w, "  %2d  %s\n", i+1, r.Title)
		fmt.Fprintf(w, "      %s\n", r.Link)
		if r.Snippet != "" {
			fmt.Fprintf(w, "      %s\n", r.Snippet)
		}
		if r.Source != "" || r.Date != "" {
			parts := []string{}
			if r.Source != "" {
				parts = append(parts, r.Source)
			}
			if r.Date != "" {
				parts = append(parts, r.Date)
			}
			fmt.Fprintf(w, "      %s\n", strings.Join(parts, " \u00b7 "))
		}
		fmt.Fprintln(w)
	}
	return nil
}

func printNewsSearchTable(cmd *cobra.Command, _ string, result *api.NewsSearchResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTITLE\tSOURCE\tDATE")
	for i, r := range result.News {
		fmt.Fprintf(w, "%2d\t%s\t%s\t%s\n", i+1, truncate(r.Title, 40), r.Source, r.Date)
	}
	return w.Flush()
}
