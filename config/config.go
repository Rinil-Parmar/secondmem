package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Model         ModelConfig         `mapstructure:"model"`
	OpenAI        OpenAIConfig        `mapstructure:"openai"`
	Ollama        OllamaConfig        `mapstructure:"ollama"`
	Copilot       CopilotConfig       `mapstructure:"copilot"`
	KnowledgeBase KnowledgeBaseConfig `mapstructure:"knowledge_base"`
	Graph         GraphConfig         `mapstructure:"graph"`
	Git           GitConfig           `mapstructure:"git"`
}

type CopilotConfig struct {
	GithubToken string `mapstructure:"github_token"`
	Model       string `mapstructure:"model"`
}

type ModelConfig struct {
	Provider string `mapstructure:"provider"`
}

type OpenAIConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type OllamaConfig struct {
	URL   string `mapstructure:"url"`
	Model string `mapstructure:"model"`
}

type KnowledgeBaseConfig struct {
	Path           string `mapstructure:"path"`
	MaxFileLines   int    `mapstructure:"max_file_lines"`
	AutoRebalance  bool   `mapstructure:"auto_rebalance"`
}

type GraphConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	DBPath  string `mapstructure:"db_path"`
}

type GitConfig struct {
	AutoCommit bool `mapstructure:"auto_commit"`
}

// DefaultBasePath returns the default base path for secondmem data.
func DefaultBasePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".secondmem"
	}
	return filepath.Join(home, ".secondmem")
}

// Load reads the config from the given file path, or the default location.
func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		basePath := DefaultBasePath()
		viper.AddConfigPath(basePath)
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	// Set defaults — Ollama is the default (no API key needed)
	viper.SetDefault("model.provider", "ollama")
	viper.SetDefault("openai.api_key", "")
	viper.SetDefault("openai.model", "gpt-4o")
	viper.SetDefault("ollama.url", "http://localhost:11434")
	viper.SetDefault("ollama.model", "llama3.2")
	viper.SetDefault("copilot.github_token", "")
	viper.SetDefault("copilot.model", "gpt-4o-mini")
	viper.SetDefault("knowledge_base.path", filepath.Join(DefaultBasePath(), "knowledge"))
	viper.SetDefault("knowledge_base.max_file_lines", 1116)
	viper.SetDefault("knowledge_base.auto_rebalance", true)
	viper.SetDefault("graph.enabled", true)
	viper.SetDefault("graph.db_path", filepath.Join(DefaultBasePath(), "secondmem.db"))
	viper.SetDefault("git.auto_commit", false)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// Expand tilde in paths
	cfg.KnowledgeBase.Path = expandPath(cfg.KnowledgeBase.Path)
	cfg.Graph.DBPath = expandPath(cfg.Graph.DBPath)

	return &cfg, nil
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
