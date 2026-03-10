package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tableCmd = &cobra.Command{
	Use:   "table",
	Short: "Manage tables",
	Long:  `Create, list, view, edit, and delete tables with typed columns and rows.`,
}

var tableListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List tables",
	Aliases: []string{"ls"},
	Example: `  fm table list
  fm table list -L 10 -p 2
  fm table list -o json`,
	RunE: runTableList,
}

var tableCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a table",
	Example: `  fm table create --name tasks
  fm table create --name tasks --description "Task tracker"
  fm table create --name tasks --columns '[{"name":"status","type":"text","isRequired":true}]'`,
	Args: cobra.NoArgs,
	RunE: runTableCreate,
}

var tableViewCmd = &cobra.Command{
	Use:   "view <table-id>",
	Short: "View a table",
	Example: `  fm table view <table-id>
  fm table view <table-id> -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runTableView,
}

var tableEditCmd = &cobra.Command{
	Use:   "edit <table-id>",
	Short: "Edit a table",
	Long:  `Edit a table's name and/or description. At least one of --name or --description must be provided.`,
	Example: `  fm table edit <table-id> --name "New Name"
  fm table edit <table-id> --description "Updated description"`,
	Args: cobra.ExactArgs(1),
	RunE: runTableEdit,
}

var tableDeleteCmd = &cobra.Command{
	Use:     "delete <table-id>",
	Short:   "Delete a table",
	Long:    `Move a table to trash. Use "table trash" to list trashed tables and "table restore" to recover them.`,
	Example: `  fm table delete <table-id>`,
	Args:    cobra.ExactArgs(1),
	RunE:    runTableDelete,
}

var tableTrashCmd = &cobra.Command{
	Use:   "trash",
	Short: "Manage tables in trash",
	Long:  `List, view, restore, and permanently delete trashed tables. Running without a subcommand lists trashed tables.`,
	RunE:  runTableTrash,
}

var tableTrashListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List tables in trash",
	Aliases: []string{"ls"},
	RunE:    runTableTrash,
}

var tableTrashViewCmd = &cobra.Command{
	Use:   "view <table-id>",
	Short: "View a trashed table",
	Args:  cobra.ExactArgs(1),
	RunE:  runTableTrashView,
}

var tableTrashRestoreCmd = &cobra.Command{
	Use:   "restore <table-id>",
	Short: "Restore a table from trash",
	Args:  cobra.ExactArgs(1),
	RunE:  runTableRestore,
}

var tableTrashDeleteCmd = &cobra.Command{
	Use:   "delete <table-id>",
	Short: "Permanently delete a trashed table",
	Long:  `Permanently delete a table from trash. This action is irreversible.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTableTrashDelete,
}

var tableRestoreCmd = &cobra.Command{
	Use:   "restore <table-id>",
	Short: "Restore a table from trash",
	Args:  cobra.ExactArgs(1),
	RunE:  runTableRestore,
}

func init() {
	tableListCmd.Flags().IntP("limit", "L", 30, "maximum number of tables to list")
	tableListCmd.Flags().IntP("page", "p", 1, "page number")

	tableCreateCmd.Flags().StringP("name", "n", "", "table name")
	tableCreateCmd.Flags().StringP("description", "d", "", "table description")
	tableCreateCmd.Flags().StringP("columns", "c", "", "columns as JSON array")
	tableCreateCmd.MarkFlagRequired("name")

	tableEditCmd.Flags().StringP("name", "n", "", "new name")
	tableEditCmd.Flags().StringP("description", "d", "", "new description")

	tableTrashCmd.Flags().IntP("limit", "L", 30, "maximum number of tables to list")
	tableTrashListCmd.Flags().IntP("limit", "L", 30, "maximum number of tables to list")

	tableTrashCmd.AddCommand(tableTrashListCmd)
	tableTrashCmd.AddCommand(tableTrashViewCmd)
	tableTrashCmd.AddCommand(tableTrashRestoreCmd)
	tableTrashCmd.AddCommand(tableTrashDeleteCmd)

	tableCmd.AddCommand(tableListCmd)
	tableCmd.AddCommand(tableCreateCmd)
	tableCmd.AddCommand(tableViewCmd)
	tableCmd.AddCommand(tableEditCmd)
	tableCmd.AddCommand(tableDeleteCmd)
	tableCmd.AddCommand(tableTrashCmd)
	tableCmd.AddCommand(tableRestoreCmd)

	rootCmd.AddCommand(tableCmd)
}

func runTableList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")

	list, err := client.ListTables(cmd.Context(), page, limit)
	if err != nil {
		return fmt.Errorf("listing tables: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printTableListTable(cmd, list)
	case "text", "":
		return printTableListText(cmd, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printTableListText(cmd *cobra.Command, list *api.TableListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No tables found.")
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d tables\n", len(list.Items), list.Total)
	for _, t := range list.Items {
		fmt.Fprintf(w, "%s  %s  %d cols  %s\n", t.ID, truncate(t.Name, 30), len(t.Columns), t.CreatedAt.Format("2006-01-02 15:04"))
	}
	return nil
}

func printTableListTable(cmd *cobra.Command, list *api.TableListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCOLUMNS\tCREATED")
	for _, t := range list.Items {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", t.ID, truncate(t.Name, 30), len(t.Columns), t.CreatedAt.Format("2006-01-02 15:04"))
	}
	return w.Flush()
}

func runTableCreate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	columnsJSON, _ := cmd.Flags().GetString("columns")

	req := &api.CreateTableRequest{
		Name:        name,
		Description: description,
	}

	if columnsJSON != "" {
		var cols []*api.CreateColumnInput
		if err := json.Unmarshal([]byte(columnsJSON), &cols); err != nil {
			return fmt.Errorf("parsing --columns JSON: %w", err)
		}
		req.Columns = cols
	}

	table, err := client.CreateTable(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Table created: %s (id=%s)\n", table.Name, table.ID)
		return nil
	}
}

func runTableView(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	table, err := client.GetTable(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("getting table: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	case "table":
		return printTableViewTable(cmd, table)
	case "text", "":
		return printTableViewText(cmd, table)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printTableViewText(cmd *cobra.Command, table *api.Table) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "ID:          %s\n", table.ID)
	fmt.Fprintf(w, "Name:        %s\n", table.Name)
	fmt.Fprintf(w, "Description: %s\n", table.Description)
	fmt.Fprintf(w, "Created:     %s\n", table.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:     %s\n", table.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(table.Columns) > 0 {
		fmt.Fprintf(w, "\nColumns (%d):\n", len(table.Columns))
		for _, col := range table.Columns {
			req := ""
			if col.IsRequired {
				req = " (required)"
			}
			fmt.Fprintf(w, "  %s  %s  %s%s\n", col.ID, col.Name, col.Type, req)
		}
	}
	return nil
}

func printTableViewTable(cmd *cobra.Command, table *api.Table) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FIELD\tVALUE")
	fmt.Fprintf(w, "ID\t%s\n", table.ID)
	fmt.Fprintf(w, "Name\t%s\n", table.Name)
	fmt.Fprintf(w, "Description\t%s\n", table.Description)
	fmt.Fprintf(w, "Created\t%s\n", table.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated\t%s\n", table.UpdatedAt.Format("2006-01-02 15:04:05"))
	if err := w.Flush(); err != nil {
		return err
	}

	if len(table.Columns) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nColumns:\n")
		cw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(cw, "ID\tNAME\tTYPE\tREQUIRED")
		for _, col := range table.Columns {
			fmt.Fprintf(cw, "%s\t%s\t%s\t%v\n", col.ID, col.Name, col.Type, col.IsRequired)
		}
		return cw.Flush()
	}
	return nil
}

func runTableEdit(cmd *cobra.Command, args []string) error {
	var patch api.TablePatch
	if cmd.Flags().Changed("name") {
		n, _ := cmd.Flags().GetString("name")
		patch.Name = &n
	}
	if cmd.Flags().Changed("description") {
		d, _ := cmd.Flags().GetString("description")
		patch.Description = &d
	}

	if patch.Name == nil && patch.Description == nil {
		return fmt.Errorf("at least one of --name or --description must be provided")
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	table, err := client.PatchTable(cmd.Context(), args[0], &patch)
	if err != nil {
		return fmt.Errorf("updating table: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Table updated: %s (id=%s)\n", table.Name, table.ID)
		return nil
	}
}

func runTableDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteTable(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("deleting table: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"id": args[0], "status": "deleted"})
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Table deleted: %s\n", args[0])
		return nil
	}
}

func runTableTrash(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")

	list, err := client.ListTrashedTables(cmd.Context(), 1, limit)
	if err != nil {
		return fmt.Errorf("listing trashed tables: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printTableTrashTable(cmd, list)
	case "text", "":
		return printTableTrashText(cmd, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printTableTrashText(cmd *cobra.Command, list *api.TableListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(w, "Trash is empty.")
		return nil
	}
	fmt.Fprintf(w, "Showing %d of %d trashed tables\n\n", len(list.Items), list.Total)
	for _, t := range list.Items {
		deletedAt := ""
		if t.DeletedAt != nil {
			deletedAt = t.DeletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "  %s  %s  %s  %d cols\n", t.ID, deletedAt, truncate(t.Name, 30), len(t.Columns))
	}
	return nil
}

func printTableTrashTable(cmd *cobra.Command, list *api.TableListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDELETED\tNAME\tCOLUMNS")
	for _, t := range list.Items {
		deletedAt := ""
		if t.DeletedAt != nil {
			deletedAt = t.DeletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", t.ID, deletedAt, truncate(t.Name, 30), len(t.Columns))
	}
	return w.Flush()
}

func runTableTrashView(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	table, err := client.GetTrashedTable(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("getting trashed table: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	case "table":
		return printTableViewTable(cmd, table)
	case "text", "":
		return printTableViewText(cmd, table)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func runTableRestore(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	table, err := client.RestoreTable(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("restoring table: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Table restored: %s (id=%s)\n", table.Name, table.ID)
		return nil
	}
}

func runTableTrashDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.PermanentlyDeleteTable(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("permanently deleting table: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"id": args[0], "status": "permanently deleted"})
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Table permanently deleted: %s\n", args[0])
		return nil
	}
}

// printTableColumns prints the column list from a table (used by field commands).
func printTableColumns(cmd *cobra.Command, table *api.Table) error {
	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	case "table":
		return printTableViewTable(cmd, table)
	case "text", "":
		return printTableViewText(cmd, table)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

// columnTypeNames returns valid column type names for help text.
func columnTypeNames() string {
	return strings.Join([]string{"text", "number", "boolean", "date", "select", "multiSelect", "url", "email", "json"}, ", ")
}
