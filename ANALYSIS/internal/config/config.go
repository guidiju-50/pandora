// Package config provides configuration management for the ANALYSIS module.
package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the ANALYSIS module.
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Quantification QuantConfig        `mapstructure:"quantification"`
	R             RConfig             `mapstructure:"r"`
	Analysis      AnalysisConfig      `mapstructure:"analysis"`
	Control       ControlAPIConfig    `mapstructure:"control"`
	Directories   DirectoriesConfig   `mapstructure:"directories"`
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// QuantConfig holds quantification tools configuration.
type QuantConfig struct {
	DefaultTool string         `mapstructure:"default_tool"`
	Threads     int            `mapstructure:"threads"`
	RSEM        RSEMConfig     `mapstructure:"rsem"`
	Kallisto    KallistoConfig `mapstructure:"kallisto"`
	Salmon      SalmonConfig   `mapstructure:"salmon"`
}

// RSEMConfig holds RSEM configuration.
type RSEMConfig struct {
	Path       string `mapstructure:"path"`
	Bowtie2Path string `mapstructure:"bowtie2_path"`
}

// KallistoConfig holds Kallisto configuration.
type KallistoConfig struct {
	Path      string `mapstructure:"path"`
	Bootstrap int    `mapstructure:"bootstrap"`
}

// SalmonConfig holds Salmon configuration.
type SalmonConfig struct {
	Path string `mapstructure:"path"`
}

// RConfig holds R configuration.
type RConfig struct {
	Path        string        `mapstructure:"path"`
	LibsPath    string        `mapstructure:"libs_path"`
	ScriptsPath string        `mapstructure:"scripts_path"`
	Timeout     time.Duration `mapstructure:"timeout"`
	MemoryLimit string        `mapstructure:"memory_limit"`
}

// AnalysisConfig holds analysis thresholds.
type AnalysisConfig struct {
	PValueThreshold  float64 `mapstructure:"pvalue_threshold"`
	Log2FCThreshold  float64 `mapstructure:"log2fc_threshold"`
	MinCountFilter   int     `mapstructure:"min_count_filter"`
}

// ControlAPIConfig holds CONTROL module API configuration.
type ControlAPIConfig struct {
	URL     string        `mapstructure:"url"`
	Timeout time.Duration `mapstructure:"timeout"`
	APIKey  string        `mapstructure:"api_key"`
}

// DirectoriesConfig holds directory paths.
type DirectoriesConfig struct {
	Data    string `mapstructure:"data"`
	Results string `mapstructure:"results"`
	Temp    string `mapstructure:"temp"`
}

// Load loads configuration from file and environment variables.
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	viper.AutomaticEnv()
	bindEnvVariables()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setDefaults() {
	// Server
	viper.SetDefault("server.port", 8082)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "300s") // Long timeout for analysis

	// Quantification
	viper.SetDefault("quantification.default_tool", "kallisto")
	viper.SetDefault("quantification.threads", 8)
	viper.SetDefault("quantification.kallisto.bootstrap", 100)

	// R
	viper.SetDefault("r.path", "/usr/bin/Rscript")
	viper.SetDefault("r.scripts_path", "./r_scripts")
	viper.SetDefault("r.timeout", "3600s")
	viper.SetDefault("r.memory_limit", "8G")

	// Analysis
	viper.SetDefault("analysis.pvalue_threshold", 0.05)
	viper.SetDefault("analysis.log2fc_threshold", 1.0)
	viper.SetDefault("analysis.min_count_filter", 10)

	// Control API
	viper.SetDefault("control.url", "http://localhost:8080")
	viper.SetDefault("control.timeout", "30s")

	// Directories
	viper.SetDefault("directories.data", "/data/analysis")
	viper.SetDefault("directories.results", "/data/results")
	viper.SetDefault("directories.temp", "/tmp/analysis")
}

func bindEnvVariables() {
	viper.BindEnv("quantification.rsem.path", "RSEM_PATH")
	viper.BindEnv("quantification.kallisto.path", "KALLISTO_PATH")
	viper.BindEnv("quantification.salmon.path", "SALMON_PATH")
	viper.BindEnv("r.path", "R_PATH")
	viper.BindEnv("r.libs_path", "R_LIBS_USER")
	viper.BindEnv("control.url", "CONTROL_API_URL")
	viper.BindEnv("control.api_key", "CONTROL_API_KEY")
}
