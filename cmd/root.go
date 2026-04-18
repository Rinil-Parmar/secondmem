package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "secondmem",
	Short: "AI-powered local knowledge management CLI",
	Long:  "secondmem is your second memory — an AI-powered CLI that ingests, organizes, and retrieves your personal knowledge using plain markdown files.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.secondmem/config.toml)")
}
