// Package config provides configuration management for the CONTROL module.
package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the CONTROL module.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	Environment  string        `mapstructure:"environment"`
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Name         string        `mapstructure:"name"`
	User         string        `mapstructure:"user"`
	Password     string        `mapstructure:"password"`
	SSLMode      string        `mapstructure:"ssl_mode"`
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

// RabbitMQConfig holds RabbitMQ configuration.
type RabbitMQConfig struct {
	URL    string       `mapstructure:"url"`
	Queues QueuesConfig `mapstructure:"queues"`
}

// QueuesConfig holds queue names configuration.
type QueuesConfig struct {
	Processing    string `mapstructure:"processing"`
	Analysis      string `mapstructure:"analysis"`
	Notifications string `mapstructure:"notifications"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret        string        `mapstructure:"secret"`
	Expiry        time.Duration `mapstructure:"expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
	Issuer        string        `mapstructure:"issuer"`
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
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
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.environment", "development")

	// Database
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "pandora")
	viper.SetDefault("database.user", "pandora")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.max_lifetime", "5m")

	// RabbitMQ
	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.queues.processing", "pandora.processing")
	viper.SetDefault("rabbitmq.queues.analysis", "pandora.analysis")
	viper.SetDefault("rabbitmq.queues.notifications", "pandora.notifications")

	// JWT
	viper.SetDefault("jwt.expiry", "24h")
	viper.SetDefault("jwt.refresh_expiry", "168h")
	viper.SetDefault("jwt.issuer", "pandora")

	// CORS
	viper.SetDefault("cors.allowed_origins", []string{"http://localhost:3000"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Authorization", "Content-Type"})
}

func bindEnvVariables() {
	viper.BindEnv("server.port", "PORT")
	viper.BindEnv("server.environment", "ENV")
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("rabbitmq.url", "RABBITMQ_URL")
	viper.BindEnv("jwt.secret", "JWT_SECRET")
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + string(rune(c.Port)) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Name +
		" sslmode=" + c.SSLMode
}
