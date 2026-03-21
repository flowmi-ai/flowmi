package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Submit support tickets",
}

var ticketCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a support ticket",
	Long: `Create a support ticket (bug report, feature request, or support question).

Ticket types:
  bug       Bug Report
  feature   Feature Request
  question  Help/Support`,
	Example: `  flowmi ticket create -t bug -s "Login broken" -m "Cannot log in on Safari"
  flowmi ticket create -t feature -s "Dark mode" -m "Please add dark mode support"
  flowmi ticket create -t question -s "How to export?" -m "How do I export my notes?"`,
	RunE: runTicketCreate,
}

var validTicketTypes = []string{"bug", "feature", "question"}

func init() {
	ticketCreateCmd.Flags().StringP("type", "t", "", "ticket type: {bug|feature|question} (required)")
	ticketCreateCmd.Flags().StringP("subject", "s", "", "ticket subject (required)")
	ticketCreateCmd.Flags().StringP("message", "m", "", "ticket message (required)")
	ticketCreateCmd.MarkFlagRequired("type")
	ticketCreateCmd.MarkFlagRequired("subject")
	ticketCreateCmd.MarkFlagRequired("message")

	ticketCmd.AddCommand(ticketCreateCmd)
	rootCmd.AddCommand(ticketCmd)
}

func runTicketCreate(cmd *cobra.Command, args []string) error {
	ticketType, _ := cmd.Flags().GetString("type")
	subject, _ := cmd.Flags().GetString("subject")
	message, _ := cmd.Flags().GetString("message")

	if err := validateEnum("type", ticketType, validTicketTypes); err != nil {
		return err
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	ticket, err := client.CreateTicket(cmd.Context(), ticketType, subject, message)
	if err != nil {
		return fmt.Errorf("creating ticket: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(ticket)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Ticket created: %s\n", ticket.ID)
	fmt.Fprintf(w, "  Type:    %s\n", formatTicketType(ticket.Type))
	fmt.Fprintf(w, "  Subject: %s\n", ticket.Subject)
	fmt.Fprintf(w, "  Status:  %s\n", ticket.Status)
	return nil
}

func formatTicketType(t string) string {
	switch t {
	case "bug":
		return "Bug Report"
	case "feature":
		return "Feature Request"
	case "question":
		return "Help/Support"
	default:
		return t
	}
}
