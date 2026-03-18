package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Generate and edit images with AI",
}

var imageGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate an image from a text prompt",
	Long: `Generate an image from a text prompt using AI.

Optionally provide reference images for editing or transformation.
The generated image is saved to a file by default.`,
	Example: `  flowmi image generate -p "A cat wearing a top hat, oil painting"
  flowmi image generate -p "Remove the background" -i photo.jpg
  flowmi image generate -p "Blend these two" -i a.jpg -i b.jpg --aspect-ratio 16:9
  flowmi image generate -p "Hi-res landscape" --size 4K --model gemini-3-pro-image-preview
  flowmi image generate -p "A logo" -o logo.png`,
	RunE: runImageGenerate,
}

func init() {
	imageGenerateCmd.Flags().StringP("prompt", "p", "", "text description of the desired image (required)")
	imageGenerateCmd.Flags().StringSliceP("image", "i", nil, "reference image path (repeatable, max 14)")
	imageGenerateCmd.Flags().StringP("model", "m", "", "model: {gemini-3.1-flash-image-preview|gemini-3-pro-image-preview|grok-imagine-image|grok-imagine-image-pro} (default \"gemini-3.1-flash-image-preview\")")
	imageGenerateCmd.Flags().StringP("aspect-ratio", "a", "", "output aspect ratio: {auto|1:1|2:3|3:2|3:4|4:3|4:5|5:4|9:16|16:9|21:9|1:4|4:1|1:8|8:1|2:1|1:2|19.5:9|9:19.5|20:9|9:20} (default \"auto\")")
	imageGenerateCmd.Flags().StringP("size", "s", "", "output resolution: {512|1K|1k|2K|2k|4K} (default \"1K\")")
	imageGenerateCmd.Flags().StringP("output", "o", "", "output file path (default: generated_<timestamp>.<ext>)")
	imageGenerateCmd.MarkFlagRequired("prompt")

	imageCmd.AddCommand(imageGenerateCmd)
	rootCmd.AddCommand(imageCmd)
}

func runImageGenerate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}
	// Image generation can be slow; extend timeout.
	client.HTTPClient.SetTimeout(120 * time.Second)

	prompt, _ := cmd.Flags().GetString("prompt")
	imagePaths, _ := cmd.Flags().GetStringSlice("image")
	model, _ := cmd.Flags().GetString("model")
	aspectRatio, _ := cmd.Flags().GetString("aspect-ratio")
	size, _ := cmd.Flags().GetString("size")

	// Validate enum flags.
	if model != "" {
		if err := validateEnum("model", model, validModels); err != nil {
			return err
		}
	}
	if aspectRatio != "" {
		if err := validateEnum("aspect-ratio", aspectRatio, validAspectRatios); err != nil {
			return err
		}
	}
	if size != "" {
		if err := validateEnum("size", size, validSizes); err != nil {
			return err
		}
	}

	// Build request.
	req := &api.ImageGenerateRequest{
		Prompt:      prompt,
		Model:       model,
		AspectRatio: stripAuto(aspectRatio),
		ImageSize:   size,
	}

	// Encode reference images.
	for _, path := range imagePaths {
		ref, err := encodeImageFile(path)
		if err != nil {
			return err
		}
		req.Images = append(req.Images, ref)
	}

	result, err := client.GenerateImage(cmd.Context(), req)
	if err != nil {
		return err
	}

	if viper.GetBool("json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	return saveAndPrintImage(cmd, result)
}

// encodeImageFile reads an image file and returns a ReferenceImage with base64 data.
func encodeImageFile(path string) (*api.ReferenceImage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading image %s: %w", path, err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = "image/jpeg" // sensible default
	}

	return &api.ReferenceImage{
		Data:     base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
	}, nil
}

// saveAndPrintImage decodes the base64 image, writes it to a file, and prints the result.
func saveAndPrintImage(cmd *cobra.Command, result *api.ImageGenerateResponse) error {
	data, err := base64.StdEncoding.DecodeString(result.Image)
	if err != nil {
		return fmt.Errorf("decoding image data: %w", err)
	}

	outFile, _ := cmd.Flags().GetString("output")
	if outFile == "" {
		ext := extFromMime(result.MimeType)
		outFile = fmt.Sprintf("generated_%s%s", time.Now().Format("20060102_150405"), ext)
	}

	if err := os.WriteFile(outFile, data, 0o644); err != nil {
		return fmt.Errorf("writing image file: %w", err)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Image saved to %s\n", outFile)
	if result.Text != "" {
		fmt.Fprintf(w, "%s\n", result.Text)
	}
	return nil
}

var validModels = []string{"gemini-3.1-flash-image-preview", "gemini-3-pro-image-preview", "grok-imagine-image", "grok-imagine-image-pro"}
var validAspectRatios = []string{"auto", "1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9", "1:4", "4:1", "1:8", "8:1", "2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20"}
var validSizes = []string{"512", "1K", "1k", "2K", "2k", "4K"}

// stripAuto returns "" for "auto", passthrough otherwise.
func stripAuto(v string) string {
	if v == "auto" {
		return ""
	}
	return v
}

func validateEnum(flag, value string, allowed []string) error {
	for _, v := range allowed {
		if v == value {
			return nil
		}
	}
	return fmt.Errorf("invalid value %q for --%s: valid values are {%s}", value, flag, strings.Join(allowed, "|"))
}

// extFromMime returns a file extension for a MIME type.
func extFromMime(mimeType string) string {
	switch strings.TrimSpace(mimeType) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/heic":
		return ".heic"
	case "image/heif":
		return ".heif"
	default:
		return ".png"
	}
}
