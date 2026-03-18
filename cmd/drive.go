package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

func normalizePath(p string) string {
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%d B (%.1f GB)", bytes, float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%d B (%.1f MB)", bytes, float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%d B (%.1f KB)", bytes, float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func detectMIMEType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return mime.TypeByExtension(ext)
}

var driveCmd = &cobra.Command{
	Use:   "drive",
	Short: "Manage your files",
	Long:  `Upload, download, list, view, and delete files in your drive.`,
}

var driveListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List files",
	Aliases: []string{"ls"},
	Example: `  fm drive list
  fm drive list -L 10 -p 2
  fm drive list --prefix /reports
  fm drive list --json`,
	RunE: runDriveList,
}

var driveUploadCmd = &cobra.Command{
	Use:   "upload [<file>]",
	Short: "Upload a file",
	Long:  `Upload a local file or stdin to your drive. When reading from stdin, --path is required.`,
	Example: `  fm drive upload ./report.pdf
  fm drive upload ./data.csv --path /reports/2024/data.csv
  echo "hello" | fm drive upload --path /notes/hello.txt
  fm drive upload ./image.png --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDriveUpload,
}

var driveDownloadCmd = &cobra.Command{
	Use:   "download <path-or-id>",
	Short: "Download a file",
	Long:  `Download a file to stdout (default) or to a local file with --dest.`,
	Example: `  fm drive download /docs/readme.txt
  fm drive download /images/photo.png -D ./photo.png
  fm drive download 550e8400-e29b-41d4-a716-446655440000 -D ./file.bin`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveDownload,
}

var driveViewCmd = &cobra.Command{
	Use:   "view <path-or-id>",
	Short: "View file metadata",
	Example: `  fm drive view /docs/readme.txt
  fm drive view 550e8400-e29b-41d4-a716-446655440000
  fm drive view /docs/readme.txt --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveView,
}

var driveDeleteCmd = &cobra.Command{
	Use:   "delete <path-or-id>",
	Short: "Delete a file",
	Long:  `Move a file to trash. Use "drive trash" to list trashed files and "drive restore" to recover them.`,
	Example: `  fm drive delete /docs/readme.txt
  fm drive delete 550e8400-e29b-41d4-a716-446655440000
  fm drive delete /docs/readme.txt --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveDelete,
}

var driveTrashCmd = &cobra.Command{
	Use:   "trash",
	Short: "Manage files in trash",
	Long:  `List, view, download, restore, and permanently delete trashed files. Running without a subcommand lists trashed files.`,
	RunE:  runDriveTrash,
}

var driveTrashListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List files in trash",
	Aliases: []string{"ls"},
	RunE:    runDriveTrash,
}

var driveTrashViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View trashed file metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveTrashView,
}

var driveTrashDownloadCmd = &cobra.Command{
	Use:   "download <id>",
	Short: "Download a trashed file",
	Long:  `Download a file from trash to stdout (default) or to a local file with --dest.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveTrashDownload,
}

var driveTrashRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a file from trash",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveRestore,
}

var driveTrashDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Permanently delete a trashed file",
	Long:  `Permanently delete a file from trash. This removes the file from storage and is irreversible.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveTrashDelete,
}

var driveRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a file from trash",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveRestore,
}

func init() {
	driveListCmd.Flags().IntP("limit", "L", 30, "maximum number of files to list")
	driveListCmd.Flags().IntP("page", "p", 1, "page number")
	driveListCmd.Flags().String("prefix", "", "filter by path prefix")

	driveUploadCmd.Flags().String("path", "", "remote path (required for stdin)")
	driveUploadCmd.Flags().String("mime-type", "", "MIME type override")

	driveDownloadCmd.Flags().StringP("dest", "D", "", "destination file path")

	driveTrashCmd.Flags().IntP("limit", "L", 30, "maximum number of files to list")
	driveTrashListCmd.Flags().IntP("limit", "L", 30, "maximum number of files to list")
	driveTrashDownloadCmd.Flags().StringP("dest", "D", "", "destination file path")

	driveTrashCmd.AddCommand(driveTrashListCmd)
	driveTrashCmd.AddCommand(driveTrashViewCmd)
	driveTrashCmd.AddCommand(driveTrashDownloadCmd)
	driveTrashCmd.AddCommand(driveTrashRestoreCmd)
	driveTrashCmd.AddCommand(driveTrashDeleteCmd)

	driveCmd.AddCommand(driveListCmd)
	driveCmd.AddCommand(driveUploadCmd)
	driveCmd.AddCommand(driveDownloadCmd)
	driveCmd.AddCommand(driveViewCmd)
	driveCmd.AddCommand(driveDeleteCmd)
	driveCmd.AddCommand(driveTrashCmd)
	driveCmd.AddCommand(driveRestoreCmd)

	rootCmd.AddCommand(driveCmd)
}

func runDriveList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	prefix, _ := cmd.Flags().GetString("prefix")

	list, err := client.ListDriveObjects(cmd.Context(), page, limit, prefix)
	if err != nil {
		return fmt.Errorf("listing files: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}
	return printDriveListText(cmd, list)
}

func printDriveListText(cmd *cobra.Command, list *api.DriveListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No files found.")
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d files\n", len(list.Items), list.Total)
	for _, obj := range list.Items {
		fmt.Fprintf(w, "%s  %s  %s  %s\n", obj.ID, obj.UpdatedAt.Format("2006-01-02 15:04"), formatSize(obj.SizeBytes), obj.Path)
	}
	return nil
}

func runDriveUpload(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	remotePath, _ := cmd.Flags().GetString("path")
	mimeType, _ := cmd.Flags().GetString("mime-type")

	var data []byte
	var filename string

	if len(args) == 1 {
		// Read from file
		filename = args[0]
		data, err = os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		if remotePath == "" {
			remotePath = "/" + filepath.Base(filename)
		}
	} else {
		// Read from stdin
		if remotePath == "" {
			return fmt.Errorf("--path is required when reading from stdin")
		}
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		filename = filepath.Base(remotePath)
	}

	remotePath = normalizePath(remotePath)

	if mimeType == "" {
		mimeType = detectMIMEType(filename)
	}

	size := int64(len(data))

	// Step 1: Init upload
	initResp, err := client.InitUpload(cmd.Context(), &api.InitUploadRequest{
		Path:      remotePath,
		SizeBytes: size,
		MimeType:  mimeType,
	})
	if err != nil {
		return fmt.Errorf("initiating upload: %w", err)
	}

	// Step 2: Upload to presigned URL
	etag, err := client.UploadToPresignedURL(cmd.Context(), initResp.UploadURL, mimeType, bytes.NewReader(data), size)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}

	// Step 3: Complete upload
	obj, err := client.CompleteUpload(cmd.Context(), initResp.ID, etag, size)
	if err != nil {
		return fmt.Errorf("completing upload: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Uploaded: %s  size=%s  id=%s\n", obj.Path, formatSize(obj.SizeBytes), obj.ID)
	return nil
}

func runDriveDownload(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	arg := args[0]
	dest, _ := cmd.Flags().GetString("dest")

	// When saving to a file, fetch metadata for MIME type info
	var obj *api.DriveObject
	if dest != "" {
		if isUUID(arg) {
			obj, err = client.GetDriveObject(cmd.Context(), arg)
		} else {
			obj, err = client.GetDriveObjectByPath(cmd.Context(), normalizePath(arg))
		}
		if err != nil {
			return fmt.Errorf("getting file metadata: %w", err)
		}
	}

	var dlResp *api.DownloadResponse
	if isUUID(arg) {
		dlResp, err = client.GetDownloadURL(cmd.Context(), arg)
	} else {
		dlResp, err = client.GetDownloadURLByPath(cmd.Context(), normalizePath(arg))
	}
	if err != nil {
		return fmt.Errorf("getting download URL: %w", err)
	}

	body, err := client.DownloadFromURL(cmd.Context(), dlResp.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer body.Close()

	if dest != "" {
		f, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}
		defer f.Close()

		n, err := io.Copy(f, body)
		if err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		mimeInfo := ""
		if obj != nil && obj.MimeType != "" {
			mimeInfo = "  type=" + obj.MimeType
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Downloaded %s to %s%s\n", formatSize(n), dest, mimeInfo)
		return nil
	}

	// Write to stdout — no status message to avoid corrupting piped data
	_, err = io.Copy(cmd.OutOrStdout(), body)
	return err
}

func runDriveView(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	arg := args[0]
	var obj *api.DriveObject
	if isUUID(arg) {
		obj, err = client.GetDriveObject(cmd.Context(), arg)
	} else {
		obj, err = client.GetDriveObjectByPath(cmd.Context(), normalizePath(arg))
	}
	if err != nil {
		return fmt.Errorf("getting file: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	}
	return printDriveViewText(cmd, obj)
}

func printDriveViewText(cmd *cobra.Command, obj *api.DriveObject) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "ID:         %s\n", obj.ID)
	fmt.Fprintf(w, "Path:       %s\n", obj.Path)
	fmt.Fprintf(w, "Size:       %s\n", formatSize(obj.SizeBytes))
	fmt.Fprintf(w, "MIME type:  %s\n", obj.MimeType)
	fmt.Fprintf(w, "Visibility: %s\n", obj.Visibility)
	fmt.Fprintf(w, "ETag:       %s\n", obj.ETag)
	fmt.Fprintf(w, "Created:    %s\n", obj.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:    %s\n", obj.UpdatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

func runDriveTrash(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")

	list, err := client.ListTrashedDriveObjects(cmd.Context(), 1, limit)
	if err != nil {
		return fmt.Errorf("listing trashed files: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}
	return printDriveTrashText(cmd, list)
}

func printDriveTrashText(cmd *cobra.Command, list *api.DriveListResponse) error {
	w := cmd.OutOrStdout()
	if len(list.Items) == 0 {
		fmt.Fprintln(w, "Trash is empty.")
		return nil
	}
	fmt.Fprintf(w, "Showing %d of %d trashed files\n\n", len(list.Items), list.Total)
	for _, obj := range list.Items {
		deletedAt := ""
		if obj.DeletedAt != nil {
			deletedAt = obj.DeletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "  %s  %s  %s  %s\n", obj.ID, deletedAt, formatSize(obj.SizeBytes), obj.Path)
	}
	return nil
}

func runDriveTrashView(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	obj, err := client.GetTrashedDriveObject(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("getting trashed file: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	}
	return printDriveViewText(cmd, obj)
}

func runDriveTrashDownload(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	dest, _ := cmd.Flags().GetString("dest")

	dlResp, err := client.GetTrashedDownloadURL(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("getting download URL: %w", err)
	}

	body, err := client.DownloadFromURL(cmd.Context(), dlResp.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer body.Close()

	if dest != "" {
		f, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}
		defer f.Close()

		n, err := io.Copy(f, body)
		if err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Downloaded %s to %s\n", formatSize(n), dest)
		return nil
	}

	_, err = io.Copy(cmd.OutOrStdout(), body)
	return err
}

func runDriveRestore(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	obj, err := client.RestoreDriveObject(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("restoring file: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "File restored: %s (id=%s)\n", obj.Path, obj.ID)
	return nil
}

func runDriveTrashDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	if err := client.PermanentlyDeleteDriveObject(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("permanently deleting file: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"id": args[0], "status": "permanently deleted"})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "File permanently deleted: %s\n", args[0])
	return nil
}

func runDriveDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	arg := args[0]
	var obj *api.DriveObject
	if isUUID(arg) {
		obj, err = client.DeleteDriveObject(cmd.Context(), arg)
	} else {
		obj, err = client.DeleteDriveObjectByPath(cmd.Context(), normalizePath(arg))
	}
	if err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Deleted: %s (id=%s)\n", obj.Path, obj.ID)
	return nil
}
