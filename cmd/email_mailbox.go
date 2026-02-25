package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/flowmi/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var emailMailboxCmd = &cobra.Command{
	Use:   "mailbox",
	Short: "Manage mailboxes",
	Long:  `Create, list, edit, and delete mailboxes.`,
}

var emailMailboxListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List mailboxes",
	Aliases: []string{"ls"},
	Example: `  fm email mailbox list
  fm email mailbox list -o json`,
	RunE: runEmailMailboxList,
}

var emailMailboxCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a mailbox",
	Example: `  fm email mailbox create --address hello@example.com
  fm email mailbox create -a hello@example.com -n "Hello Mailbox"`,
	Args: cobra.NoArgs,
	RunE: runEmailMailboxCreate,
}

var emailMailboxEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a mailbox",
	Long:  `Edit a mailbox's display name and/or active status. At least one of --display-name or --active must be provided.`,
	Example: `  fm email mailbox edit <id> --display-name "New Name"
  fm email mailbox edit <id> --active=false`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailMailboxEdit,
}

var emailMailboxDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete a mailbox",
	Example: `  fm email mailbox delete <id>`,
	Args:    cobra.ExactArgs(1),
	RunE:    runEmailMailboxDelete,
}

func init() {
	emailMailboxCreateCmd.Flags().StringP("address", "a", "", "mailbox email address")
	emailMailboxCreateCmd.Flags().StringP("display-name", "n", "", "display name")
	emailMailboxCreateCmd.MarkFlagRequired("address")

	emailMailboxEditCmd.Flags().StringP("display-name", "n", "", "new display name")
	emailMailboxEditCmd.Flags().Bool("active", true, "set active status")

	emailMailboxCmd.AddCommand(emailMailboxListCmd)
	emailMailboxCmd.AddCommand(emailMailboxCreateCmd)
	emailMailboxCmd.AddCommand(emailMailboxEditCmd)
	emailMailboxCmd.AddCommand(emailMailboxDeleteCmd)

	emailCmd.AddCommand(emailMailboxCmd)
}

func runEmailMailboxList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	mailboxes, err := client.ListMailboxes(cmd.Context())
	if err != nil {
		return fmt.Errorf("listing mailboxes: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(mailboxes)
	case "table":
		return printMailboxListTable(cmd, mailboxes)
	case "text", "":
		return printMailboxListText(cmd, mailboxes)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printMailboxListText(cmd *cobra.Command, mailboxes []*api.Mailbox) error {
	w := cmd.OutOrStdout()
	if len(mailboxes) == 0 {
		fmt.Fprintln(w, "No mailboxes found.")
		return nil
	}
	fmt.Fprintf(w, "Showing %d mailboxes\n\n", len(mailboxes))
	for _, m := range mailboxes {
		active := "active"
		if !m.IsActive {
			active = "inactive"
		}
		fmt.Fprintf(w, "  %s  %s  %s  %s\n", m.ID, m.Address, m.DisplayName, active)
	}
	return nil
}

func printMailboxListTable(cmd *cobra.Command, mailboxes []*api.Mailbox) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tADDRESS\tDISPLAY NAME\tACTIVE\tCREATED")
	for _, m := range mailboxes {
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", m.ID, m.Address, m.DisplayName, m.IsActive, m.CreatedAt.Format("2006-01-02 15:04"))
	}
	return w.Flush()
}

func runEmailMailboxCreate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	address, _ := cmd.Flags().GetString("address")
	displayName, _ := cmd.Flags().GetString("display-name")

	req := &api.CreateMailboxRequest{
		Address:     address,
		DisplayName: displayName,
	}

	mailbox, err := client.CreateMailbox(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("creating mailbox: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(mailbox)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Mailbox created: %s (%s)\n", mailbox.Address, mailbox.ID)
		return nil
	}
}

func runEmailMailboxEdit(cmd *cobra.Command, args []string) error {
	var patch api.MailboxPatch
	hasChange := false

	if cmd.Flags().Changed("display-name") {
		n, _ := cmd.Flags().GetString("display-name")
		patch.DisplayName = &n
		hasChange = true
	}
	if cmd.Flags().Changed("active") {
		a, _ := cmd.Flags().GetBool("active")
		patch.IsActive = &a
		hasChange = true
	}

	if !hasChange {
		return fmt.Errorf("at least one of --display-name or --active must be provided")
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	mailbox, err := client.PatchMailbox(cmd.Context(), args[0], &patch)
	if err != nil {
		return fmt.Errorf("updating mailbox: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(mailbox)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Mailbox updated: %s (%s)\n", mailbox.Address, mailbox.ID)
		return nil
	}
}

func runEmailMailboxDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteMailbox(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("deleting mailbox: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"id": args[0], "status": "deleted"})
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Mailbox deleted: %s\n", args[0])
		return nil
	}
}
