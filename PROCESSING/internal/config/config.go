// Package config provides configuration management for the PROCESSING module.
package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the PROCESSING module.
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Scraper     ScraperConfig     `mapstructure:"scraper"`
	Trimmomatic TrimmoConfig      `mapstructure:"trimmomatic"`
	ETL         ETLConfig         `mapstructure:"etl"`
	Control     ControlAPIConfig  `mapstructure:"control"`
	Directories DirectoriesConfig `mapstructure:"directories"`
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// ScraperConfig holds web scraping configuration.
type ScraperConfig struct {
	NCBI NCBIConfig `mapstructure:"ncbi"`
}

// NCBIConfig holds NCBI API configuration.
type NCBIConfig struct {
	APIKey      string        `mapstructure:"api_key"`
	BaseURL     string        `mapstructure:"base_url"`
	RateLimit   int           `mapstructure:"rate_limit"`
	Timeout     time.Duration `mapstructure:"timeout"`
	MaxRetries  int           `mapstructure:"max_retries"`
	RetryDelay  time.Duration `mapstructure:"retry_delay"`
}

// TrimmoConfig holds Trimmomatic configuration.
type TrimmoConfig struct {
	JarPath       string `mapstructure:"jar_path"`
	AdaptersPath  string `mapstructure:"adapters_path"`
	Threads       int    `mapstructure:"threads"`
	Leading       int    `mapstructure:"leading"`
	Trailing      int    `mapstructure:"trailing"`
	SlidingWindow string `mapstructure:"sliding_window"`
	MinLen        int    `mapstructure:"min_len"`
}

// ETLConfig holds ETL pipeline configuration.
type ETLConfig struct {
	BatchSize     int `mapstructure:"batch_size"`
	RetryAttempts int `mapstructure:"retry_attempts"`
	WorkerCount   int `mapstructure:"worker_count"`
}

// ControlAPIConfig holds CONTROL module API configuration.
type ControlAPIConfig struct {
	URL     string        `mapstructure:"url"`
	Timeout time.Duration `mapstructure:"timeout"`
	APIKey  string        `mapstructure:"api_key"`
}

// DirectoriesConfig holds directory paths configuration.
type DirectoriesConfig struct {
	Data   string `mapstructure:"data"`
	Temp   string `mapstructure:"temp"`
	Output string `mapstructure:"output"`
}

// Load loads configuration from file and environment variables.
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Override with environment variables
	viper.AutomaticEnv()
	bindEnvVariables()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets default configuration values.
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8081)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")

	// NCBI defaults
	viper.SetDefault("scraper.ncbi.base_url", "https://eutils.ncbi.nlm.nih.gov/entrez/eutils")
	viper.SetDefault("scraper.ncbi.rate_limit", 3)
	viper.SetDefault("scraper.ncbi.timeout", "30s")
	viper.SetDefault("scraper.ncbi.max_retries", 3)
	viper.SetDefault("scraper.ncbi.retry_delay", "1s")

	// Trimmomatic defaults
	viper.SetDefault("trimmomatic.threads", 4)
	viper.SetDefault("trimmomatic.leading", 3)
	viper.SetDefault("trimmomatic.trailing", 3)
	viper.SetDefault("trimmomatic.sliding_window", "4:15")
	viper.SetDefault("trimmomatic.min_len", 36)

	// ETL defaults
	viper.SetDefault("etl.batch_size", 1000)
	viper.SetDefault("etl.retry_attempts", 3)
	viper.SetDefault("etl.worker_count", 4)

	// Control API defaults
	viper.SetDefault("control.url", "http://localhost:8080")
	viper.SetDefault("control.timeout", "30s")

	// Directory defaults
	viper.SetDefault("directories.data", "/data/processing")
	viper.SetDefault("directories.temp", "/tmp/processing")
	viper.SetDefault("directories.output", "/data/output")
}

// bindEnvVariables binds environment variables to config keys.
func bindEnvVariables() {
	viper.BindEnv("scraper.ncbi.api_key", "NCBI_API_KEY")
	viper.BindEnv("trimmomatic.jar_path", "TRIMMOMATIC_JAR")
	viper.BindEnv("trimmomatic.adapters_path", "TRIMMOMATIC_ADAPTERS")
	viper.BindEnv("control.url", "CONTROL_API_URL")
	viper.BindEnv("control.api_key", "CONTROL_API_KEY")
	viper.BindEnv("directories.data", "DATA_DIR")
	viper.BindEnv("directories.temp", "TEMP_DIR")
	viper.BindEnv("directories.output", "OUTPUT_DIR")
}
