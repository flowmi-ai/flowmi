package cmd

import (
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
	Use:   "flowmi",
	Short: "Your all-in-one command-line tool",
	Long: `flowmi (fm) brings stock quotes, weather, news, AI image/video generation,
and more to your terminal. One tool, endless possibilities.`,
}

func Execute() {
	// Support "fm" as alias: adapt Use field based on binary name
	bin := filepath.Base(os.Args[0])
	if bin == "fm" {
		rootCmd.Use = "fm"
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
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
