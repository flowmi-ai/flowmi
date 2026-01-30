package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.flowmi.yaml)")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "output format: text, json, table")
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".flowmi")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("FLOWMI")
	viper.AutomaticEnv()
	viper.ReadInConfig() // silently ignore if config file not found
}
