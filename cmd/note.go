package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/flowmi/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// truncate truncates s to maxRunes runes, appending "..." if truncated.
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage your notes",
	Long:  `Create, list, view, edit, and delete your notes.`,
}

var noteListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all notes",
	Aliases: []string{"ls"},
	RunE:    runNoteList,
}

var noteCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new note",
	Aliases: []string{"new"},
	Args:    cobra.NoArgs,
	RunE:    runNoteCreate,
}

var noteViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a note",
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteView,
}

var noteEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a note",
	Long:  `Edit a note's subject and/or content. At least one of --subject or --content must be provided.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteEdit,
}

var noteDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteDelete,
}

func init() {
	noteListCmd.Flags().IntP("limit", "L", 30, "maximum number of notes to list")

	noteCreateCmd.Flags().StringP("subject", "s", "", "note subject")
	noteCreateCmd.Flags().StringP("content", "c", "", "note content")
	noteCreateCmd.MarkFlagRequired("content")

	noteEditCmd.Flags().StringP("subject", "s", "", "new subject")
	noteEditCmd.Flags().StringP("content", "c", "", "new content")

	noteDeleteCmd.Flags().Bool("yes", false, "skip confirmation prompt")

	noteCmd.AddCommand(noteListCmd)
	noteCmd.AddCommand(noteCreateCmd)
	noteCmd.AddCommand(noteViewCmd)
	noteCmd.AddCommand(noteEditCmd)
	noteCmd.AddCommand(noteDeleteCmd)

	rootCmd.AddCommand(noteCmd)
}

func newNoteClient() (*api.Client, error) {
	accessToken := viper.GetString("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("not logged in. run 'flowmi auth login' to get started")
	}
	apiServerURL := viper.GetString("api_server_url")
	return api.NewClient(apiServerURL, accessToken), nil
}

func runNoteList(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")

	list, err := client.ListNotes(cmd.Context(), 1, limit)
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
	fmt.Fprintf(w, "Showing %d of %d notes\n\n", len(list.Items), list.Total)
	for _, n := range list.Items {
		fmt.Fprintf(w, "  %s  %s  %s  %s\n", n.ID, n.CreatedAt.Format("2006-01-02 15:04"), truncate(n.Subject, 30), truncate(n.Content, 50))
	}
	return nil
}

func printNoteListTable(cmd *cobra.Command, list *api.NoteListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCREATED\tSUBJECT\tCONTENT")
	for _, n := range list.Items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.ID, n.CreatedAt.Format("2006-01-02 15:04"), truncate(n.Subject, 30), truncate(n.Content, 40))
	}
	return w.Flush()
}

func runNoteCreate(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	subject, _ := cmd.Flags().GetString("subject")
	content, _ := cmd.Flags().GetString("content")

	note, err := client.CreateNote(cmd.Context(), subject, content)
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

func runNoteView(cmd *cobra.Command, args []string) error {
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
	fmt.Fprintf(w, "Subject:  %s\n", note.Subject)
	fmt.Fprintf(w, "Created:  %s\n", note.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:  %s\n", note.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "\n%s\n", note.Content)
	return nil
}

func printNoteTable(cmd *cobra.Command, note *api.Note) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FIELD\tVALUE")
	fmt.Fprintf(w, "ID\t%s\n", note.ID)
	fmt.Fprintf(w, "Subject\t%s\n", note.Subject)
	fmt.Fprintf(w, "Created\t%s\n", note.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated\t%s\n", note.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Content\t%s\n", note.Content)
	return w.Flush()
}

func runNoteEdit(cmd *cobra.Command, args []string) error {
	fields := make(map[string]string)
	if cmd.Flags().Changed("subject") {
		subject, _ := cmd.Flags().GetString("subject")
		fields["subject"] = subject
	}
	if cmd.Flags().Changed("content") {
		content, _ := cmd.Flags().GetString("content")
		fields["content"] = content
	}

	if len(fields) == 0 {
		return fmt.Errorf("at least one of --subject or --content must be provided")
	}

	client, err := newNoteClient()
	if err != nil {
		return err
	}

	note, err := client.PatchNote(cmd.Context(), args[0], fields)
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
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Fprintf(cmd.OutOrStdout(), "? Delete note %s? (y/N) ", args[0])
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		answer = strings.TrimSpace(answer)
		if answer != "y" && answer != "Y" {
			return fmt.Errorf("cancelled")
		}
	}

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
