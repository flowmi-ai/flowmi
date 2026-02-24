package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flowmi/flowmi/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

const (
	defaultAuthServerURL = "https://auth.flowmi.ai"
	defaultAPIServerURL  = "https://api.flowmi.ai"
)

var rootCmd = &cobra.Command{
	Use:           "flowmi",
	Short:         "Flowmi CLI",
	Long:          `flowmi (fm) — notes, auth, and more from your terminal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	// Support "fm" as alias: adapt Use field based on binary name
	bin := filepath.Base(os.Args[0])
	if bin == "fm" {
		rootCmd.Use = "fm"
	}

	if err := rootCmd.Execute(); err != nil {
		output := viper.GetString("output")
		if output == "json" {
			je := struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}{
				Error:   "command_error",
				Message: err.Error(),
			}
			_ = json.NewEncoder(os.Stderr).Encode(je)
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/flowmi/config.toml)")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "output format: text, json, table")
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := config.ConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetDefault("auth_server_url", defaultAuthServerURL)
	viper.SetDefault("api_server_url", defaultAPIServerURL)
	viper.SetEnvPrefix("FLOWMI")
	viper.AutomaticEnv()
	viper.ReadInConfig() // silently ignore if config file not found

	// Inject credentials into viper so the rest of the app can use
	// viper.GetString("access_token") etc. as before.
	creds, err := config.LoadCredentials()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not load credentials:", err)
		return
	}
	for k, v := range creds {
		viper.SetDefault(k, v)
	}
}
