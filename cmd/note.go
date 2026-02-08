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

// truncate truncates s to maxRunes runes, appending "..." if truncated.
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}

// formatLabels formats a label slice for display.
func formatLabels(labels []string) string {
	if len(labels) == 0 {
		return "(none)"
	}
	return strings.Join(labels, ", ")
}

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage your notes",
	Long:  `Create, list, view, edit, delete, trash, and restore your notes.`,
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
	Long:  `Edit a note's subject, content, and/or labels. At least one of --subject, --content, or --label must be provided.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteEdit,
}

var noteDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a note",
	Long:  `Move a note to trash. Use "note trash" to list trashed notes and "note restore" to recover them.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteDelete,
}

var noteTrashCmd = &cobra.Command{
	Use:   "trash",
	Short: "List notes in trash",
	RunE:  runNoteTrash,
}

var noteRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a note from trash",
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteRestore,
}

func init() {
	noteListCmd.Flags().IntP("limit", "L", 30, "maximum number of notes to list")
	noteListCmd.Flags().StringSliceP("label", "l", nil, "filter by label (repeatable, comma-separated)")

	noteCreateCmd.Flags().StringP("subject", "s", "", "note subject")
	noteCreateCmd.Flags().StringP("content", "c", "", "note content")
	noteCreateCmd.Flags().StringSliceP("label", "l", nil, "label (repeatable, comma-separated)")
	noteCreateCmd.MarkFlagRequired("content")

	noteEditCmd.Flags().StringP("subject", "s", "", "new subject")
	noteEditCmd.Flags().StringP("content", "c", "", "new content")
	noteEditCmd.Flags().StringSliceP("label", "l", nil, "set labels (repeatable, comma-separated, replaces all)")

	noteTrashCmd.Flags().IntP("limit", "L", 30, "maximum number of notes to list")

	noteCmd.AddCommand(noteListCmd)
	noteCmd.AddCommand(noteCreateCmd)
	noteCmd.AddCommand(noteViewCmd)
	noteCmd.AddCommand(noteEditCmd)
	noteCmd.AddCommand(noteDeleteCmd)
	noteCmd.AddCommand(noteTrashCmd)
	noteCmd.AddCommand(noteRestoreCmd)

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
	labels, _ := cmd.Flags().GetStringSlice("label")

	list, err := client.ListNotes(cmd.Context(), 1, limit, labels, "")
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
	labels, _ := cmd.Flags().GetStringSlice("label")

	note, err := client.CreateNote(cmd.Context(), subject, content, labels)
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
	fmt.Fprintf(w, "Labels:   %s\n", formatLabels(note.Labels))
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
	fmt.Fprintf(w, "Labels\t%s\n", formatLabels(note.Labels))
	fmt.Fprintf(w, "Created\t%s\n", note.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated\t%s\n", note.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Content\t%s\n", note.Content)
	return w.Flush()
}

func runNoteEdit(cmd *cobra.Command, args []string) error {
	var patch api.NotePatch
	if cmd.Flags().Changed("subject") {
		s, _ := cmd.Flags().GetString("subject")
		patch.Subject = &s
	}
	if cmd.Flags().Changed("content") {
		c, _ := cmd.Flags().GetString("content")
		patch.Content = &c
	}
	if cmd.Flags().Changed("label") {
		labels, _ := cmd.Flags().GetStringSlice("label")
		// --label "" clears all labels
		if len(labels) == 1 && labels[0] == "" {
			labels = []string{}
		}
		patch.Labels = &labels
	}

	if patch.Subject == nil && patch.Content == nil && patch.Labels == nil {
		return fmt.Errorf("at least one of --subject, --content, or --label must be provided")
	}

	client, err := newNoteClient()
	if err != nil {
		return err
	}

	note, err := client.PatchNote(cmd.Context(), args[0], &patch)
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

	note, err := client.DeleteNote(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("deleting note: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(note)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Note deleted: %s\n", note.ID)
		return nil
	}
}

func runNoteTrash(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")

	list, err := client.ListNotes(cmd.Context(), 1, limit, nil, "trashed")
	if err != nil {
		return fmt.Errorf("listing trashed notes: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printTrashTable(cmd, list)
	case "text", "":
		return printTrashText(cmd, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printTrashText(cmd *cobra.Command, list *api.NoteListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(w, "Trash is empty.")
		return nil
	}
	fmt.Fprintf(w, "Showing %d of %d trashed notes\n\n", len(list.Items), list.Total)
	for _, n := range list.Items {
		deletedAt := ""
		if n.DeletedAt != nil {
			deletedAt = n.DeletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "  %s  %s  %s  %s\n", n.ID, deletedAt, truncate(n.Subject, 30), truncate(n.Content, 50))
	}
	return nil
}

func printTrashTable(cmd *cobra.Command, list *api.NoteListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDELETED\tSUBJECT\tCONTENT")
	for _, n := range list.Items {
		deletedAt := ""
		if n.DeletedAt != nil {
			deletedAt = n.DeletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.ID, deletedAt, truncate(n.Subject, 30), truncate(n.Content, 40))
	}
	return w.Flush()
}

func runNoteRestore(cmd *cobra.Command, args []string) error {
	client, err := newNoteClient()
	if err != nil {
		return err
	}

	note, err := client.RestoreNote(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("restoring note: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(note)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Note restored: %s\n", note.ID)
		return nil
	}
}
