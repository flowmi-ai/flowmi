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

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Manage emails",
	Long:  `Send, list, view, and delete emails. Manage mailboxes.`,
}

var emailListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List emails",
	Aliases: []string{"ls"},
	Example: `  fm email list
  fm email list -L 10 -p 2
  fm email list --direction inbound
  fm email list -o json`,
	RunE: runEmailList,
}

var emailViewCmd = &cobra.Command{
	Use:     "view <id>",
	Short:   "View an email",
	Example: `  fm email view <id>`,
	Args:    cobra.ExactArgs(1),
	RunE:    runEmailView,
}

var emailSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an email",
	Example: `  fm email send --mailbox <id> --to user@example.com --subject "Hello" --text "Hi there"
  fm email send -m <id> -t user@example.com -s "Hello" --html "<h1>Hi</h1>"`,
	Args: cobra.NoArgs,
	RunE: runEmailSend,
}

var emailDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete an email",
	Example: `  fm email delete <id>`,
	Args:    cobra.ExactArgs(1),
	RunE:    runEmailDelete,
}

func init() {
	emailListCmd.Flags().IntP("limit", "L", 30, "maximum number of emails to list")
	emailListCmd.Flags().IntP("page", "p", 1, "page number")
	emailListCmd.Flags().StringP("direction", "d", "", "filter by direction (inbound, outbound)")

	emailSendCmd.Flags().StringP("mailbox", "m", "", "mailbox ID")
	emailSendCmd.Flags().StringSliceP("to", "t", nil, "recipient address (repeatable, comma-separated)")
	emailSendCmd.Flags().StringSlice("cc", nil, "CC address (repeatable, comma-separated)")
	emailSendCmd.Flags().StringSlice("bcc", nil, "BCC address (repeatable, comma-separated)")
	emailSendCmd.Flags().String("reply-to", "", "reply-to address")
	emailSendCmd.Flags().StringP("subject", "s", "", "email subject")
	emailSendCmd.Flags().String("text", "", "plain text body")
	emailSendCmd.Flags().String("html", "", "HTML body")
	emailSendCmd.MarkFlagRequired("to")
	emailSendCmd.MarkFlagRequired("subject")

	emailCmd.AddCommand(emailListCmd)
	emailCmd.AddCommand(emailViewCmd)
	emailCmd.AddCommand(emailSendCmd)
	emailCmd.AddCommand(emailDeleteCmd)

	rootCmd.AddCommand(emailCmd)
}

func runEmailList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	direction, _ := cmd.Flags().GetString("direction")

	list, err := client.ListEmails(cmd.Context(), page, limit, direction)
	if err != nil {
		return fmt.Errorf("listing emails: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case "table":
		return printEmailListTable(cmd, list)
	case "text", "":
		return printEmailListText(cmd, list)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printEmailListText(cmd *cobra.Command, list *api.EmailListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(w, "No emails found.")
		return nil
	}
	fmt.Fprintf(w, "Showing %d of %d emails\n\n", len(list.Items), list.Total)
	for _, e := range list.Items {
		sentAt := ""
		if e.SentAt != nil {
			sentAt = e.SentAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "  %s  %s  %s  %s  %s\n", e.ID, sentAt, e.Direction, e.From, truncate(e.Subject, 40))
	}
	return nil
}

func printEmailListTable(cmd *cobra.Command, list *api.EmailListResponse) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSENT\tDIRECTION\tFROM\tTO\tSUBJECT\tSTATUS")
	for _, e := range list.Items {
		sentAt := ""
		if e.SentAt != nil {
			sentAt = e.SentAt.Format("2006-01-02 15:04")
		}
		to := strings.Join(e.To, ", ")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", e.ID, sentAt, e.Direction, e.From, truncate(to, 30), truncate(e.Subject, 30), e.Status)
	}
	return w.Flush()
}

func runEmailView(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	email, err := client.GetEmail(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("getting email: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(email)
	case "table":
		return printEmailViewTable(cmd, email)
	case "text", "":
		return printEmailViewText(cmd, email)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printEmailViewText(cmd *cobra.Command, email *api.EmailDetail) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "ID:        %s\n", email.ID)
	fmt.Fprintf(w, "Direction: %s\n", email.Direction)
	fmt.Fprintf(w, "From:      %s\n", email.From)
	fmt.Fprintf(w, "To:        %s\n", strings.Join(email.To, ", "))
	if len(email.CC) > 0 {
		fmt.Fprintf(w, "CC:        %s\n", strings.Join(email.CC, ", "))
	}
	fmt.Fprintf(w, "Subject:   %s\n", email.Subject)
	fmt.Fprintf(w, "Status:    %s\n", email.Status)
	if email.SentAt != nil {
		fmt.Fprintf(w, "Sent:      %s\n", email.SentAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(w, "Created:   %s\n", email.CreatedAt.Format("2006-01-02 15:04:05"))

	if email.TextBody != "" {
		fmt.Fprintf(w, "\n%s\n", email.TextBody)
	} else if email.HTMLBody != "" {
		fmt.Fprintln(w, "\n(HTML only — use -o json to see full content)")
	}
	return nil
}

func printEmailViewTable(cmd *cobra.Command, email *api.EmailDetail) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FIELD\tVALUE")
	fmt.Fprintf(w, "ID\t%s\n", email.ID)
	fmt.Fprintf(w, "Direction\t%s\n", email.Direction)
	fmt.Fprintf(w, "From\t%s\n", email.From)
	fmt.Fprintf(w, "To\t%s\n", strings.Join(email.To, ", "))
	if len(email.CC) > 0 {
		fmt.Fprintf(w, "CC\t%s\n", strings.Join(email.CC, ", "))
	}
	fmt.Fprintf(w, "Subject\t%s\n", email.Subject)
	fmt.Fprintf(w, "Status\t%s\n", email.Status)
	if email.SentAt != nil {
		fmt.Fprintf(w, "Sent\t%s\n", email.SentAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(w, "Created\t%s\n", email.CreatedAt.Format("2006-01-02 15:04:05"))
	return w.Flush()
}

func runEmailSend(cmd *cobra.Command, args []string) error {
	textBody, _ := cmd.Flags().GetString("text")
	htmlBody, _ := cmd.Flags().GetString("html")
	if textBody == "" && htmlBody == "" {
		return fmt.Errorf("at least one of --text or --html must be provided")
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	mailboxID, _ := cmd.Flags().GetString("mailbox")
	to, _ := cmd.Flags().GetStringSlice("to")
	cc, _ := cmd.Flags().GetStringSlice("cc")
	bcc, _ := cmd.Flags().GetStringSlice("bcc")
	replyTo, _ := cmd.Flags().GetString("reply-to")
	subject, _ := cmd.Flags().GetString("subject")

	req := &api.SendEmailRequest{
		MailboxID: mailboxID,
		To:        to,
		CC:        cc,
		BCC:       bcc,
		ReplyTo:   replyTo,
		Subject:   subject,
		HTML:      htmlBody,
		Text:      textBody,
	}

	email, err := client.SendEmail(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(email)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Email sent: %s\n", email.ID)
		return nil
	}
}

func runEmailDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteEmail(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("deleting email: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"id": args[0], "status": "deleted"})
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Email deleted: %s\n", args[0])
		return nil
	}
}
