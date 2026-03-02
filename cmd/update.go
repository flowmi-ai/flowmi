package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	"github.com/flowmi/flowmi/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type updateResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Updated        bool   `json:"updated"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update flowmi to the latest version",
	Long:  "Check for and install the latest version of flowmi from GitHub Releases.",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	updateCmd.Flags().Bool("dry-run", false, "Check for updates without installing")
	updateCmd.Flags().String("version", "", "Update to a specific version (e.g. v0.2.0)")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	targetVersion, _ := cmd.Flags().GetString("version")

	output := viper.GetString("output")
	switch output {
	case "json", "text", "table", "":
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}

	// Refuse to update dev builds.
	if version == "dev" {
		return fmt.Errorf("cannot update a dev build; install from a release first")
	}

	// Normalize --version to match GitHub tag format (e.g. "1.2.0" → "v1.2.0").
	if targetVersion != "" && !strings.HasPrefix(targetVersion, "v") {
		targetVersion = "v" + targetVersion
	}

	// Set up GitHub source + checksum validator.
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return fmt.Errorf("failed to create update source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	// Detect release.
	repo := selfupdate.ParseSlug("flowmi-ai/flowmi")
	var release *selfupdate.Release
	var found bool

	if targetVersion != "" {
		release, found, err = updater.DetectVersion(cmd.Context(), repo, targetVersion)
	} else {
		release, found, err = updater.DetectLatest(cmd.Context(), repo)
	}
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if !found {
		if output == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(updateResult{
				CurrentVersion: version,
			})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No releases found.")
		return nil
	}

	// Already up to date?
	if release.LessOrEqual(version) {
		if output == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(updateResult{
				CurrentVersion: version,
				LatestVersion:  release.Version(),
				Updated:        false,
			})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", ui.InfoStyle.Render("v"+version))
		fmt.Fprintln(cmd.OutOrStdout(), ui.SuccessStyle.Render("Already up to date."))
		return nil
	}

	// Show version diff.
	if output != "json" {
		fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", ui.InfoStyle.Render("v"+version))
		fmt.Fprintf(cmd.OutOrStdout(), "Latest version:  %s\n", ui.SuccessStyle.Render("v"+release.Version()))
	}

	// Dry-run: stop here.
	if dryRun {
		if output == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(updateResult{
				CurrentVersion: version,
				LatestVersion:  release.Version(),
				Updated:        false,
			})
		}
		fmt.Fprintln(cmd.OutOrStdout(), ui.SubtleStyle.Render("Run without --dry-run to install."))
		return nil
	}

	// Confirm unless --yes.
	if !yes {
		fmt.Fprintf(cmd.OutOrStdout(), "\nUpdate to v%s? [Y/n] ", release.Version())
		reader := bufio.NewReader(cmd.InOrStdin())
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != "" && answer != "y" && answer != "Y" {
			fmt.Fprintln(cmd.OutOrStdout(), "Update cancelled.")
			return nil
		}
	}

	// Download and replace.
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}

	if err := updater.UpdateTo(cmd.Context(), release, exe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if output == "json" {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(updateResult{
			CurrentVersion: version,
			LatestVersion:  release.Version(),
			Updated:        true,
		})
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%s Updated to v%s\n",
		ui.SuccessStyle.Render("Success!"), release.Version())
	return nil
}
