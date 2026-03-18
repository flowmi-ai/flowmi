package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Generate videos with AI",
}

var videoGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a video from a text prompt",
	Long: `Generate a video from a text prompt using AI.

Supports text-to-video, image-to-video (with --image), and video editing
(with --video-url). Video generation is asynchronous — the CLI polls until
the video is ready, then downloads it.`,
	Example: `  flowmi video generate -p "A rocket launching from Mars"
  flowmi video generate -p "A sunset timelapse" -d 10 -a 16:9 -r 720p
  flowmi video generate -p "Animate this scene" -i photo.jpg -d 5
  flowmi video generate -p "Change the car color to red" --video-url https://example.com/video.mp4
  flowmi video generate -p "Ocean waves" -o waves.mp4`,
	RunE: runVideoGenerate,
}

func init() {
	videoGenerateCmd.Flags().StringP("prompt", "p", "", "text description of the desired video (required)")
	videoGenerateCmd.Flags().StringP("image", "i", "", "source image path for image-to-video")
	videoGenerateCmd.Flags().String("video-url", "", "source video URL for video editing (duration capped at 8.7s)")
	videoGenerateCmd.Flags().IntP("duration", "d", 0, "video length in seconds: 1–15 (required)")
	videoGenerateCmd.Flags().StringP("model", "m", "", "model: {grok-imagine-video} (default \"grok-imagine-video\")")
	videoGenerateCmd.Flags().StringP("aspect-ratio", "a", "", "output aspect ratio: {auto|1:1|16:9|9:16|4:3|3:4|3:2|2:3} (default \"auto\")")
	videoGenerateCmd.Flags().StringP("resolution", "r", "", "output resolution: {480p|720p} (default \"480p\")")
	videoGenerateCmd.Flags().StringP("output", "o", "", "output file path (default: generated_<timestamp>.mp4)")
	videoGenerateCmd.MarkFlagRequired("prompt")
	videoGenerateCmd.MarkFlagRequired("duration")

	videoCmd.AddCommand(videoGenerateCmd)
	rootCmd.AddCommand(videoCmd)
}

var validVideoModels = []string{"grok-imagine-video"}
var validVideoAspectRatios = []string{"auto", "1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"}
var validVideoResolutions = []string{"480p", "720p"}

func runVideoGenerate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	prompt, _ := cmd.Flags().GetString("prompt")
	imagePath, _ := cmd.Flags().GetString("image")
	videoURL, _ := cmd.Flags().GetString("video-url")
	duration, _ := cmd.Flags().GetInt("duration")
	model, _ := cmd.Flags().GetString("model")
	aspectRatio, _ := cmd.Flags().GetString("aspect-ratio")
	resolution, _ := cmd.Flags().GetString("resolution")

	// Validate enum flags.
	if model != "" {
		if err := validateEnum("model", model, validVideoModels); err != nil {
			return err
		}
	}
	if aspectRatio != "" {
		if err := validateEnum("aspect-ratio", aspectRatio, validVideoAspectRatios); err != nil {
			return err
		}
	}
	if resolution != "" {
		if err := validateEnum("resolution", resolution, validVideoResolutions); err != nil {
			return err
		}
	}
	if duration < 1 || duration > 15 {
		return fmt.Errorf("invalid value %d for --duration: must be 1–15", duration)
	}

	// Build request.
	req := &api.VideoGenerateRequest{
		Prompt:      prompt,
		Model:       model,
		Duration:    duration,
		AspectRatio: stripAuto(aspectRatio),
		Resolution:  resolution,
	}

	// Encode source image as data URI for image-to-video.
	if imagePath != "" {
		dataURI, err := encodeImageDataURI(imagePath)
		if err != nil {
			return err
		}
		req.Image = &api.VideoRef{URL: dataURI}
	}

	// Source video for video editing.
	if videoURL != "" {
		req.Video = &api.VideoRef{URL: videoURL}
	}

	// Step 1: Submit generation request.
	if !viper.GetBool("json") {
		fmt.Fprintln(cmd.OutOrStdout(), "Submitting video generation request...")
	}
	genResp, err := client.GenerateVideo(cmd.Context(), req)
	if err != nil {
		return err
	}

	if viper.GetBool("json") {
		return pollAndOutputJSON(cmd, client, genResp.RequestID)
	}
	return pollAndDownload(cmd, client, genResp.RequestID)
}

func pollAndDownload(cmd *cobra.Command, client *api.Client, requestID string) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Request ID: %s\n", requestID)
	fmt.Fprintf(w, "Polling for completion (every 5s)...\n")

	for {
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-time.After(5 * time.Second):
		}

		status, err := client.GetVideoStatus(cmd.Context(), requestID)
		if err != nil {
			return err
		}

		switch status.Status {
		case "pending":
			fmt.Fprintf(w, ".")
			continue
		case "expired":
			fmt.Fprintln(w)
			return fmt.Errorf("video generation expired")
		case "done":
			fmt.Fprintln(w)
			if status.Video == nil {
				return fmt.Errorf("video ready but no URL returned")
			}

			outFile, _ := cmd.Flags().GetString("output")
			if outFile == "" {
				outFile = fmt.Sprintf("generated_%s.mp4", time.Now().Format("20060102_150405"))
			}

			if err := downloadFile(status.Video.URL, outFile); err != nil {
				return fmt.Errorf("downloading video: %w", err)
			}

			fmt.Fprintf(w, "Video saved to %s (%ds)\n", outFile, status.Video.Duration)
			return nil
		default:
			fmt.Fprintln(w)
			return fmt.Errorf("unexpected status: %s", status.Status)
		}
	}
}

func pollAndOutputJSON(cmd *cobra.Command, client *api.Client, requestID string) error {
	for {
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-time.After(5 * time.Second):
		}

		status, err := client.GetVideoStatus(cmd.Context(), requestID)
		if err != nil {
			return err
		}

		if status.Status == "pending" {
			continue
		}

		// Download file if --output is set and video is ready.
		var savedTo string
		outFile, _ := cmd.Flags().GetString("output")
		if outFile != "" && status.Status == "done" && status.Video != nil {
			if err := downloadFile(status.Video.URL, outFile); err != nil {
				return fmt.Errorf("downloading video: %w", err)
			}
			savedTo = outFile
		}

		result := struct {
			RequestID string `json:"requestId"`
			SavedTo   string `json:"savedTo,omitempty"`
			*api.VideoStatusResponse
		}{
			RequestID:           requestID,
			SavedTo:             savedTo,
			VideoStatusResponse: status,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
}

// encodeImageDataURI reads an image file and returns a base64 data URI.
func encodeImageDataURI(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading image %s: %w", path, err)
	}
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data)), nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(dest)
		return err
	}
	return f.Close()
}
