package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tableFieldCmd = &cobra.Command{
	Use:     "field",
	Short:   "Manage table columns",
	Aliases: []string{"col", "column"},
	Long:    `Add, edit, and delete columns on a table.`,
}

var tableFieldAddCmd = &cobra.Command{
	Use:   "add <table-id>",
	Short: "Add a column to a table",
	Example: `  fm table field add <table-id> --name status --type text
  fm table field add <table-id> --name priority --type number --required
  fm table field add <table-id> --name category --type select --options '{"choices":["a","b"]}'`,
	Args: cobra.ExactArgs(1),
	RunE: runTableFieldAdd,
}

var tableFieldEditCmd = &cobra.Command{
	Use:   "edit <table-id> <column-id>",
	Short: "Edit a column",
	Example: `  fm table field edit <table-id> <column-id> --name "new name"
  fm table field edit <table-id> <column-id> --type number
  fm table field edit <table-id> <column-id> --required=false`,
	Args: cobra.ExactArgs(2),
	RunE: runTableFieldEdit,
}

var tableFieldDeleteCmd = &cobra.Command{
	Use:     "delete <table-id> <column-id>",
	Short:   "Delete a column",
	Example: `  fm table field delete <table-id> <column-id>`,
	Args:    cobra.ExactArgs(2),
	RunE:    runTableFieldDelete,
}

func init() {
	tableFieldAddCmd.Flags().StringP("name", "n", "", "column name")
	tableFieldAddCmd.Flags().StringP("type", "t", "", "column type ("+columnTypeNames()+")")
	tableFieldAddCmd.Flags().BoolP("required", "r", false, "column is required")
	tableFieldAddCmd.Flags().String("options", "", "column options as JSON")
	tableFieldAddCmd.MarkFlagRequired("name")
	tableFieldAddCmd.MarkFlagRequired("type")

	tableFieldEditCmd.Flags().StringP("name", "n", "", "new column name")
	tableFieldEditCmd.Flags().StringP("type", "t", "", "new column type")
	tableFieldEditCmd.Flags().BoolP("required", "r", false, "column is required")
	tableFieldEditCmd.Flags().String("options", "", "column options as JSON")

	tableFieldCmd.AddCommand(tableFieldAddCmd)
	tableFieldCmd.AddCommand(tableFieldEditCmd)
	tableFieldCmd.AddCommand(tableFieldDeleteCmd)

	tableCmd.AddCommand(tableFieldCmd)
}

func runTableFieldAdd(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	colType, _ := cmd.Flags().GetString("type")
	required, _ := cmd.Flags().GetBool("required")
	optionsJSON, _ := cmd.Flags().GetString("options")

	input := &api.CreateColumnInput{
		Name:       name,
		Type:       colType,
		IsRequired: required,
	}

	if optionsJSON != "" {
		input.Options = json.RawMessage(optionsJSON)
	}

	table, err := client.AddColumn(cmd.Context(), args[0], input)
	if err != nil {
		return fmt.Errorf("adding column: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Column added to table %s\n", table.ID)
	return printTableColumns(cmd, table)
}

func runTableFieldEdit(cmd *cobra.Command, args []string) error {
	var patch api.ColumnPatch
	hasChange := false

	if cmd.Flags().Changed("name") {
		n, _ := cmd.Flags().GetString("name")
		patch.Name = &n
		hasChange = true
	}
	if cmd.Flags().Changed("type") {
		t, _ := cmd.Flags().GetString("type")
		patch.Type = &t
		hasChange = true
	}
	if cmd.Flags().Changed("required") {
		r, _ := cmd.Flags().GetBool("required")
		patch.IsRequired = &r
		hasChange = true
	}
	if cmd.Flags().Changed("options") {
		o, _ := cmd.Flags().GetString("options")
		patch.Options = json.RawMessage(o)
		hasChange = true
	}

	if !hasChange {
		return fmt.Errorf("at least one of --name, --type, --required, or --options must be provided")
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	table, err := client.PatchColumn(cmd.Context(), args[0], args[1], &patch)
	if err != nil {
		return fmt.Errorf("updating column: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(table)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Column updated on table %s\n", table.ID)
	return nil
}

func runTableFieldDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteColumn(cmd.Context(), args[0], args[1]); err != nil {
		return fmt.Errorf("deleting column: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"tableId": args[0], "columnId": args[1], "status": "deleted"})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Column %s deleted from table %s\n", args[1], args[0])
	return nil
}
