package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and update configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}
		fmt.Printf("Provider:       %s\n", cfg.Model.Provider)
		fmt.Printf("Ollama URL:     %s\n", cfg.Ollama.URL)
		fmt.Printf("Ollama Model:   %s\n", cfg.Ollama.Model)
		fmt.Printf("OpenAI Model:   %s\n", cfg.OpenAI.Model)
		apiKey := cfg.OpenAI.APIKey
		if len(apiKey) > 8 {
			apiKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		} else if apiKey == "" {
			apiKey = "(not set)"
		}
		fmt.Printf("OpenAI API Key: %s\n", apiKey)
		fmt.Printf("Knowledge Path: %s\n", cfg.KnowledgeBase.Path)
		fmt.Printf("Max File Lines: %d\n", cfg.KnowledgeBase.MaxFileLines)
		fmt.Printf("Auto Rebalance: %v\n", cfg.KnowledgeBase.AutoRebalance)
		fmt.Printf("Graph Enabled:  %v\n", cfg.Graph.Enabled)
		fmt.Printf("Graph DB:       %s\n", cfg.Graph.DBPath)
		fmt.Printf("Git Auto-Commit:%v\n", cfg.Git.AutoCommit)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// Load existing config
		configPath := cfgFile
		if configPath == "" {
			configPath = filepath.Join(config.DefaultBasePath(), "config.toml")
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("config file not found. Run 'secondmem init' first")
		}

		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return err
		}

		viper.Set(key, value)

		// Mask sensitive values in output
		displayValue := value
		if strings.Contains(key, "api_key") && len(value) > 8 {
			displayValue = value[:4] + "..." + value[len(value)-4:]
		}

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Set %s = %s\n", key, displayValue)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
