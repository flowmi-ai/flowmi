package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/flowmi/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage your notes",
	Long:  `Create, list, view, update, and delete your notes.`,
}

var noteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes",
	RunE:  runNoteList,
}

var noteCreateCmd = &cobra.Command{
	Use:   "create <content>",
	Short: "Create a new note",
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteCreate,
}

var noteGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a note by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteGet,
}

var noteUpdateCmd = &cobra.Command{
	Use:   "update <id> <content>",
	Short: "Update a note",
	Args:  cobra.ExactArgs(2),
	RunE:  runNoteUpdate,
}

var noteDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteDelete,
}

func init() {
	noteListCmd.Flags().Int("page", 1, "page number")
	noteListCmd.Flags().Int("page-size", 20, "items per page")

	noteCmd.AddCommand(noteListCmd)
	noteCmd.AddCommand(noteCreateCmd)
	noteCmd.AddCommand(noteGetCmd)
	noteCmd.AddCommand(noteUpdateCmd)
	noteCmd.AddCommand(noteDeleteCmd)

	rootCmd.AddCommand(noteCmd)
}

func newNoteClient() (*api.Client, error) {
	accessToken := viper.GetString("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("not logged in. run 'flowmi login' to get started")
	}
	apiServerURL := viper.GetString("api_server_url")
	return api.NewClient(apiServerURL, accessToken), nil
}

func runNoteList(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	list, err := client.ListNotes(cmd.Context(), page, pageSize)
	if err != nil {
		return fmt.Errorf("listing notes: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printNoteListTable(cmd, list)
	case "text", "":
		return printNoteListText(cmd, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printNoteListText(cmd *cobra.Command, list *api.NoteListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(w, "No notes found.")
		return nil
	}
	fmt.Fprintf(w, "Notes (page %d, %d total)\n\n", list.Page, list.Total)
	for _, n := range list.Items {
		content := n.Content
		if len(content) > 80 {
			content = content[:77] + "..."
		}
		fmt.Fprintf(w, "  %s  %s  %s\n", n.ID, n.CreatedAt.Format("2006-01-02 15:04"), content)
	}
	return nil
}

func printNoteListTable(cmd *cobra.Command, list *api.NoteListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCREATED\tCONTENT")
	for _, n := range list.Items {
		content := n.Content
		if len(content) > 60 {
			content = content[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", n.ID, n.CreatedAt.Format("2006-01-02 15:04"), content)
	}
	return w.Flush()
}

func runNoteCreate(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	note, err := client.CreateNote(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("creating note: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(note)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Note created: %s\n", note.ID)
		return nil
	}
}

func runNoteGet(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	note, err := client.GetNote(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("getting note: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(note)
	case "table":
		return printNoteTable(cmd, note)
	case "text", "":
		return printNoteText(cmd, note)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printNoteText(cmd *cobra.Command, note *api.Note) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "ID:       %s\n", note.ID)
	fmt.Fprintf(w, "Created:  %s\n", note.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:  %s\n", note.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "\n%s\n", note.Content)
	return nil
}

func printNoteTable(cmd *cobra.Command, note *api.Note) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FIELD\tVALUE")
	fmt.Fprintf(w, "ID\t%s\n", note.ID)
	fmt.Fprintf(w, "Created\t%s\n", note.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated\t%s\n", note.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Content\t%s\n", note.Content)
	return w.Flush()
}

func runNoteUpdate(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	note, err := client.UpdateNote(cmd.Context(), args[0], args[1])
	if err != nil {
		return fmt.Errorf("updating note: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(note)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Note updated: %s\n", note.ID)
		return nil
	}
}

func runNoteDelete(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	if err := client.DeleteNote(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("deleting note: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"status": "deleted", "id": args[0]})
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Note deleted: %s\n", args[0])
		return nil
	}
}
