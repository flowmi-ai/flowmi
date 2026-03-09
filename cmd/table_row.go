package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/flowmi/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tableRowCmd = &cobra.Command{
	Use:   "row",
	Short: "Manage table rows",
	Long:  `List, create, view, edit, delete, and query rows in a table.`,
}

var tableRowListCmd = &cobra.Command{
	Use:     "list <table-id>",
	Short:   "List rows",
	Aliases: []string{"ls"},
	Example: `  fm table row list <table-id>
  fm table row list <table-id> -L 10 -p 2
  fm table row list <table-id> -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runTableRowList,
}

var tableRowCreateCmd = &cobra.Command{
	Use:   "create <table-id>",
	Short: "Create a row",
	Example: `  fm table row create <table-id> --set status=todo --set priority=3
  fm table row create <table-id> --data '{"status":"todo","priority":3}'
  echo '{"status":"done"}' | fm table row create <table-id> --data -`,
	Args: cobra.ExactArgs(1),
	RunE: runTableRowCreate,
}

var tableRowViewCmd = &cobra.Command{
	Use:   "view <table-id> <row-id>",
	Short: "View a row",
	Example: `  fm table row view <table-id> <row-id>
  fm table row view <table-id> <row-id> -o json`,
	Args: cobra.ExactArgs(2),
	RunE: runTableRowView,
}

var tableRowEditCmd = &cobra.Command{
	Use:   "edit <table-id> <row-id>",
	Short: "Edit a row",
	Example: `  fm table row edit <table-id> <row-id> --set status=done
  fm table row edit <table-id> <row-id> --data '{"status":"done"}'`,
	Args: cobra.ExactArgs(2),
	RunE: runTableRowEdit,
}

var tableRowDeleteCmd = &cobra.Command{
	Use:     "delete <table-id> <row-id>",
	Short:   "Delete a row",
	Long:    `Move a row to trash. Use "table row trash" to list trashed rows and "table row restore" to recover them.`,
	Example: `  fm table row delete <table-id> <row-id>`,
	Args:    cobra.ExactArgs(2),
	RunE:    runTableRowDelete,
}

var tableRowTrashCmd = &cobra.Command{
	Use:   "trash <table-id>",
	Short: "Manage rows in trash",
	Long:  `List, view, restore, and permanently delete trashed rows. Running without a subcommand lists trashed rows.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTableRowTrash,
}

var tableRowTrashListCmd = &cobra.Command{
	Use:     "list <table-id>",
	Short:   "List rows in trash",
	Aliases: []string{"ls"},
	Args:    cobra.ExactArgs(1),
	RunE:    runTableRowTrash,
}

var tableRowTrashViewCmd = &cobra.Command{
	Use:   "view <table-id> <row-id>",
	Short: "View a trashed row",
	Args:  cobra.ExactArgs(2),
	RunE:  runTableRowTrashView,
}

var tableRowTrashRestoreCmd = &cobra.Command{
	Use:   "restore <table-id> <row-id>",
	Short: "Restore a row from trash",
	Args:  cobra.ExactArgs(2),
	RunE:  runTableRowRestore,
}

var tableRowTrashDeleteCmd = &cobra.Command{
	Use:   "delete <table-id> <row-id>",
	Short: "Permanently delete a trashed row",
	Long:  `Permanently delete a row from trash. This action is irreversible.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runTableRowTrashDelete,
}

var tableRowRestoreCmd = &cobra.Command{
	Use:   "restore <table-id> <row-id>",
	Short: "Restore a row from trash",
	Args:  cobra.ExactArgs(2),
	RunE:  runTableRowRestore,
}

var tableRowQueryCmd = &cobra.Command{
	Use:   "query <table-id>",
	Short: "Query rows with filters, sorting, aggregation, and grouping",
	Example: `  fm table row query <table-id> --filter status:eq:todo
  fm table row query <table-id> --filter priority:gt:3 --sort priority:desc
  fm table row query <table-id> --filter status:eq:done --filter priority:gte:5 -L 10
  fm table row query <table-id> -a count::total
  fm table row query <table-id> --filter status:eq:active -a count::total -a sum:amount:total_amount
  fm table row query <table-id> -a min:price:cheapest -a max:price:most_expensive
  fm table row query <table-id> -g status -a count::total
  fm table row query <table-id> -g status -a sum:revenue:total_revenue -s total_revenue:desc
  fm table row query <table-id> -g status -g region -a count::total -f region:eq:US`,
	Args: cobra.ExactArgs(1),
	RunE: runTableRowQuery,
}

func init() {
	tableRowListCmd.Flags().IntP("limit", "L", 30, "maximum number of rows to list")
	tableRowListCmd.Flags().IntP("page", "p", 1, "page number")

	tableRowCreateCmd.Flags().String("data", "", `row data as JSON (use "-" for stdin)`)
	tableRowCreateCmd.Flags().StringSlice("set", nil, "set field value (repeatable, key=value)")

	tableRowEditCmd.Flags().String("data", "", `row data as JSON (use "-" for stdin)`)
	tableRowEditCmd.Flags().StringSlice("set", nil, "set field value (repeatable, key=value)")

	tableRowQueryCmd.Flags().StringSliceP("filter", "f", nil, "filter condition (repeatable, column:op:value)")
	tableRowQueryCmd.Flags().StringSliceP("sort", "s", nil, "sort order (repeatable, column:direction)")
	tableRowQueryCmd.Flags().StringSliceP("aggregate", "a", nil, "aggregate function (repeatable, fn:column:alias)")
	tableRowQueryCmd.Flags().StringSliceP("group-by", "g", nil, "group by column (repeatable)")
	tableRowQueryCmd.Flags().IntP("limit", "L", 30, "maximum number of rows")
	tableRowQueryCmd.Flags().IntP("page", "p", 1, "page number")

	tableRowTrashCmd.Flags().IntP("limit", "L", 30, "maximum number of rows to list")
	tableRowTrashListCmd.Flags().IntP("limit", "L", 30, "maximum number of rows to list")

	tableRowTrashCmd.AddCommand(tableRowTrashListCmd)
	tableRowTrashCmd.AddCommand(tableRowTrashViewCmd)
	tableRowTrashCmd.AddCommand(tableRowTrashRestoreCmd)
	tableRowTrashCmd.AddCommand(tableRowTrashDeleteCmd)

	tableRowCmd.AddCommand(tableRowListCmd)
	tableRowCmd.AddCommand(tableRowCreateCmd)
	tableRowCmd.AddCommand(tableRowViewCmd)
	tableRowCmd.AddCommand(tableRowEditCmd)
	tableRowCmd.AddCommand(tableRowDeleteCmd)
	tableRowCmd.AddCommand(tableRowQueryCmd)
	tableRowCmd.AddCommand(tableRowTrashCmd)
	tableRowCmd.AddCommand(tableRowRestoreCmd)

	tableCmd.AddCommand(tableRowCmd)
}

// parseRowData builds a map[string]any from --data and --set flags.
// --data is parsed first, --set overlays on top.
func parseRowData(cmd *cobra.Command) (map[string]any, error) {
	data := make(map[string]any)

	dataFlag, _ := cmd.Flags().GetString("data")
	if dataFlag != "" {
		var raw []byte
		var err error
		if dataFlag == "-" {
			raw, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("reading stdin: %w", err)
			}
		} else {
			raw = []byte(dataFlag)
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, fmt.Errorf("parsing --data JSON: %w", err)
		}
	}

	setFlags, _ := cmd.Flags().GetStringSlice("set")
	for _, kv := range setFlags {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --set format %q (expected key=value)", kv)
		}
		data[parts[0]] = coerceValue(parts[1])
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("at least one of --data or --set must be provided")
	}

	return data, nil
}

// coerceValue converts a string value to the appropriate Go type.
func coerceValue(s string) any {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if s == "null" {
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// parseFilter parses "column:op:value" into a QueryCondition.
func parseFilter(s string) (*api.QueryCondition, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid filter format %q (expected column:op or column:op:value)", s)
	}
	cond := &api.QueryCondition{
		Column: parts[0],
		Op:     parts[1],
	}
	if len(parts) == 3 {
		cond.Value = coerceValue(parts[2])
	}
	return cond, nil
}

// parseSort parses "column:direction" into a QuerySort.
func parseSort(s string) (*api.QuerySort, error) {
	parts := strings.SplitN(s, ":", 2)
	sort := &api.QuerySort{
		Column: parts[0],
	}
	if len(parts) == 2 {
		sort.Direction = parts[1]
	}
	return sort, nil
}

func runTableRowList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	tableID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")

	// Fetch table schema for column names
	table, err := client.GetTable(cmd.Context(), tableID)
	if err != nil {
		return fmt.Errorf("getting table schema: %w", err)
	}

	list, err := client.ListRows(cmd.Context(), tableID, page, limit)
	if err != nil {
		return fmt.Errorf("listing rows: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printRowListTable(cmd, table, list)
	case "text", "":
		return printRowListText(cmd, table, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printRowListText(cmd *cobra.Command, table *api.Table, list *api.RowListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No rows found.")
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d rows\n", len(list.Items), list.Total)
	for _, row := range list.Items {
		fields := formatRowFields(table, row)
		fmt.Fprintf(w, "%s  %s  %s\n", row.ID, row.CreatedAt.Format("2006-01-02 15:04"), fields)
	}
	return nil
}

func printRowListTable(cmd *cobra.Command, table *api.Table, list *api.RowListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	// Build header: ID + column names + CREATED
	headers := []string{"ID"}
	for _, col := range table.Columns {
		headers = append(headers, strings.ToUpper(col.Name))
	}
	headers = append(headers, "CREATED")
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, row := range list.Items {
		parts := []string{row.ID}
		for _, col := range table.Columns {
			val := formatCellValue(row.Data[col.Name])
			parts = append(parts, truncate(val, 30))
		}
		parts = append(parts, row.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Fprintln(w, strings.Join(parts, "\t"))
	}
	return w.Flush()
}

func formatRowFields(table *api.Table, row *api.Row) string {
	var parts []string
	for _, col := range table.Columns {
		val := formatCellValue(row.Data[col.Name])
		parts = append(parts, col.Name+"="+truncate(val, 20))
	}
	return strings.Join(parts, "  ")
}

func formatCellValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func runTableRowCreate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	data, err := parseRowData(cmd)
	if err != nil {
		return err
	}

	row, err := client.CreateRow(cmd.Context(), args[0], data)
	if err != nil {
		return fmt.Errorf("creating row: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(row)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Row created: %s\n", row.ID)
		return nil
	}
}

func runTableRowView(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	row, err := client.GetRow(cmd.Context(), args[0], args[1])
	if err != nil {
		return fmt.Errorf("getting row: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(row)
	case "table":
		return printRowViewTable(cmd, row)
	case "text", "":
		return printRowViewText(cmd, row)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printRowViewText(cmd *cobra.Command, row *api.Row) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "ID:       %s\n", row.ID)
	fmt.Fprintf(w, "Table:    %s\n", row.TableID)
	fmt.Fprintf(w, "Created:  %s\n", row.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:  %s\n", row.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintln(w, "\nData:")
	for k, v := range row.Data {
		fmt.Fprintf(w, "  %s: %s\n", k, formatCellValue(v))
	}
	return nil
}

func printRowViewTable(cmd *cobra.Command, row *api.Row) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FIELD\tVALUE")
	fmt.Fprintf(w, "ID\t%s\n", row.ID)
	fmt.Fprintf(w, "Table\t%s\n", row.TableID)
	fmt.Fprintf(w, "Created\t%s\n", row.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated\t%s\n", row.UpdatedAt.Format("2006-01-02 15:04:05"))
	for k, v := range row.Data {
		fmt.Fprintf(w, "%s\t%s\n", k, formatCellValue(v))
	}
	return w.Flush()
}

func runTableRowEdit(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	data, err := parseRowData(cmd)
	if err != nil {
		return err
	}

	row, err := client.PatchRow(cmd.Context(), args[0], args[1], data)
	if err != nil {
		return fmt.Errorf("updating row: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(row)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Row updated: %s\n", row.ID)
		return nil
	}
}

func runTableRowDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteRow(cmd.Context(), args[0], args[1]); err != nil {
		return fmt.Errorf("deleting row: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"tableId": args[0], "rowId": args[1], "status": "deleted"})
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Row deleted: %s\n", args[1])
		return nil
	}
}

// parseAggregate parses "fn:column:alias" into an AggregateFunc.
func parseAggregate(s string) (*api.AggregateFunc, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid aggregate format %q (expected fn:column:alias)", s)
	}
	fn := parts[0]
	col := parts[1]
	alias := parts[2]
	if fn == "" || alias == "" {
		return nil, fmt.Errorf("invalid aggregate format %q (fn and alias are required)", s)
	}
	agg := &api.AggregateFunc{
		Fn:    fn,
		Alias: alias,
	}
	if col != "" {
		agg.Column = col
	}
	return agg, nil
}

func runTableRowQuery(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	tableID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	filters, _ := cmd.Flags().GetStringSlice("filter")
	sorts, _ := cmd.Flags().GetStringSlice("sort")
	aggregates, _ := cmd.Flags().GetStringSlice("aggregate")
	groupByCols, _ := cmd.Flags().GetStringSlice("group-by")

	req := &api.QueryRequest{
		Page:     page,
		PageSize: limit,
	}

	if len(filters) > 0 {
		var conditions []*api.QueryCondition
		for _, f := range filters {
			cond, err := parseFilter(f)
			if err != nil {
				return err
			}
			conditions = append(conditions, cond)
		}
		req.Filter = &api.QueryFilter{And: conditions}
	}

	for _, s := range sorts {
		sort, err := parseSort(s)
		if err != nil {
			return err
		}
		req.Sort = append(req.Sort, sort)
	}

	// Group-by mode: requires aggregate, returns per-group stats
	if len(groupByCols) > 0 {
		if len(aggregates) == 0 {
			return fmt.Errorf("--group-by requires at least one --aggregate function")
		}
		var aliases []string
		for _, a := range aggregates {
			agg, err := parseAggregate(a)
			if err != nil {
				return err
			}
			req.Aggregate = append(req.Aggregate, agg)
			aliases = append(aliases, agg.Alias)
		}
		req.GroupBy = groupByCols

		resp, err := client.GroupByRows(cmd.Context(), tableID, req)
		if err != nil {
			return fmt.Errorf("querying grouped rows: %w", err)
		}

		output := viper.GetString("output")
		switch output {
		case "json":
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		case "table":
			return printGroupByTable(cmd, resp, groupByCols, aliases)
		case "text", "":
			return printGroupByText(cmd, resp, groupByCols, aliases)
		default:
			return fmt.Errorf("unsupported output format: %s", output)
		}
	}

	// Aggregate mode: different response shape, skip table schema
	if len(aggregates) > 0 {
		var aliases []string
		for _, a := range aggregates {
			agg, err := parseAggregate(a)
			if err != nil {
				return err
			}
			req.Aggregate = append(req.Aggregate, agg)
			aliases = append(aliases, agg.Alias)
		}
		req.Page = 0
		req.PageSize = 0

		resp, err := client.AggregateRows(cmd.Context(), tableID, req)
		if err != nil {
			return fmt.Errorf("aggregating rows: %w", err)
		}

		output := viper.GetString("output")
		switch output {
		case "json":
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		case "table":
			return printAggregateTable(cmd, resp, aliases)
		case "text", "":
			return printAggregateText(cmd, resp, aliases)
		default:
			return fmt.Errorf("unsupported output format: %s", output)
		}
	}

	// Fetch table schema for column display
	table, err := client.GetTable(cmd.Context(), tableID)
	if err != nil {
		return fmt.Errorf("getting table schema: %w", err)
	}

	list, err := client.QueryRows(cmd.Context(), tableID, req)
	if err != nil {
		return fmt.Errorf("querying rows: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printRowListTable(cmd, table, list)
	case "text", "":
		return printRowListText(cmd, table, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printAggregateText(cmd *cobra.Command, resp *api.AggregateResponse, aliases []string) error {
	w := cmd.OutOrStdout()
	for _, k := range aliases {
		fmt.Fprintf(w, "%s: %s\n", k, formatCellValue(resp.Results[k]))
	}
	return nil
}

func printAggregateTable(cmd *cobra.Command, resp *api.AggregateResponse, aliases []string) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tVALUE")
	for _, k := range aliases {
		fmt.Fprintf(w, "%s\t%s\n", k, formatCellValue(resp.Results[k]))
	}
	return w.Flush()
}

func printGroupByText(cmd *cobra.Command, resp *api.GroupByResponse, groupCols, aliases []string) error {
	w := cmd.OutOrStdout()
	if len(resp.Groups) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No groups found.")
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d groups\n", len(resp.Groups), resp.Total)
	for i, group := range resp.Groups {
		if i > 0 {
			fmt.Fprintln(w)
		}
		for _, col := range groupCols {
			fmt.Fprintf(w, "%s: %s\n", col, formatCellValue(group[col]))
		}
		for _, alias := range aliases {
			fmt.Fprintf(w, "%s: %s\n", alias, formatCellValue(group[alias]))
		}
	}
	return nil
}

func printGroupByTable(cmd *cobra.Command, resp *api.GroupByResponse, groupCols, aliases []string) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	headers := make([]string, 0, len(groupCols)+len(aliases))
	for _, col := range groupCols {
		headers = append(headers, strings.ToUpper(col))
	}
	for _, alias := range aliases {
		headers = append(headers, strings.ToUpper(alias))
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, group := range resp.Groups {
		parts := make([]string, 0, len(groupCols)+len(aliases))
		for _, col := range groupCols {
			parts = append(parts, formatCellValue(group[col]))
		}
		for _, alias := range aliases {
			parts = append(parts, formatCellValue(group[alias]))
		}
		fmt.Fprintln(w, strings.Join(parts, "\t"))
	}
	return w.Flush()
}

func runTableRowTrash(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	tableID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	// Fetch table schema for column names
	table, err := client.GetTable(cmd.Context(), tableID)
	if err != nil {
		// Table might be trashed too, try getting trashed table
		table, err = client.GetTrashedTable(cmd.Context(), tableID)
		if err != nil {
			return fmt.Errorf("getting table schema: %w", err)
		}
	}

	list, err := client.ListTrashedRows(cmd.Context(), tableID, 1, limit)
	if err != nil {
		return fmt.Errorf("listing trashed rows: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printRowTrashTable(cmd, table, list)
	case "text", "":
		return printRowTrashText(cmd, table, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printRowTrashText(cmd *cobra.Command, table *api.Table, list *api.RowListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(w, "Trash is empty.")
		return nil
	}
	fmt.Fprintf(w, "Showing %d of %d trashed rows\n\n", len(list.Items), list.Total)
	for _, row := range list.Items {
		deletedAt := ""
		if row.DeletedAt != nil {
			deletedAt = row.DeletedAt.Format("2006-01-02 15:04")
		}
		fields := formatRowFields(table, row)
		fmt.Fprintf(w, "  %s  %s  %s\n", row.ID, deletedAt, fields)
	}
	return nil
}

func printRowTrashTable(cmd *cobra.Command, table *api.Table, list *api.RowListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	headers := []string{"ID", "DELETED"}
	for _, col := range table.Columns {
		headers = append(headers, strings.ToUpper(col.Name))
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, row := range list.Items {
		deletedAt := ""
		if row.DeletedAt != nil {
			deletedAt = row.DeletedAt.Format("2006-01-02 15:04")
		}
		parts := []string{row.ID, deletedAt}
		for _, col := range table.Columns {
			val := formatCellValue(row.Data[col.Name])
			parts = append(parts, truncate(val, 30))
		}
		fmt.Fprintln(w, strings.Join(parts, "\t"))
	}
	return w.Flush()
}

func runTableRowTrashView(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	row, err := client.GetTrashedRow(cmd.Context(), args[0], args[1])
	if err != nil {
		return fmt.Errorf("getting trashed row: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(row)
	case "table":
		return printRowViewTable(cmd, row)
	case "text", "":
		return printRowViewText(cmd, row)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func runTableRowRestore(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	row, err := client.RestoreRow(cmd.Context(), args[0], args[1])
	if err != nil {
		return fmt.Errorf("restoring row: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(row)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Row restored: %s\n", row.ID)
		return nil
	}
}

func runTableRowTrashDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.PermanentlyDeleteRow(cmd.Context(), args[0], args[1]); err != nil {
		return fmt.Errorf("permanently deleting row: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"tableId": args[0], "rowId": args[1], "status": "permanently deleted"})
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Row permanently deleted: %s\n", args[1])
		return nil
	}
}
