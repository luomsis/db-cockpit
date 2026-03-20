package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Cache     CacheConfig     `yaml:"cache"`
	MessageMQ MessageMQConfig `yaml:"message_queue"`
	Auth      AuthConfig      `yaml:"auth"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig holds server-related configurations
type ServerConfig struct {
	Gateway    ServiceConfig `yaml:"gateway"`
	Collector  ServiceConfig `yaml:"collector"`
	Agent      ServiceConfig `yaml:"agent"`
	TaskEngine ServiceConfig `yaml:"taskengine"`
	DataQuery  ServiceConfig `yaml:"dataquery"`
}

// ServiceConfig holds individual service configuration
type ServiceConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// DatabaseConfig holds database connection configurations
type DatabaseConfig struct {
	TimescaleDB TimescaleDBConfig `yaml:"timescaledb"`
	Neo4j       Neo4jConfig       `yaml:"neo4j"`
	PgVector    PgVectorConfig    `yaml:"pgvector"`
}

// TimescaleDBConfig for TimescaleDB connection
type TimescaleDBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
	MaxConns int    `yaml:"max_conns"`
	MinConns int    `yaml:"min_conns"`
}

// Neo4jConfig for Neo4j connection
type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// PgVectorConfig for pgvector connection
type PgVectorConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
}

// CacheConfig holds cache configurations
type CacheConfig struct {
	Redis RedisConfig `yaml:"redis"`
}

// RedisConfig for Redis connection
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// MessageMQConfig holds message queue configurations
type MessageMQConfig struct {
	PgMQ PgMQConfig `yaml:"pgmq"`
}

// PgMQConfig for PGMQ connection
type PgMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// AuthConfig holds authentication configurations
type AuthConfig struct {
	JWTSecret   string `yaml:"jwt_secret"`
	TokenExpire int    `yaml:"token_expire"`
}

// LoggingConfig holds logging configurations
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Gateway: ServiceConfig{
				Host: "0.0.0.0",
				Port: 8080,
				Mode: "debug",
			},
			Collector: ServiceConfig{
				Host: "0.0.0.0",
				Port: 8081,
			},
			Agent: ServiceConfig{
				Host: "0.0.0.0",
				Port: 8082,
			},
			TaskEngine: ServiceConfig{
				Host: "0.0.0.0",
				Port: 8083,
			},
			DataQuery: ServiceConfig{
				Host: "0.0.0.0",
				Port: 8084,
			},
		},
		Database: DatabaseConfig{
			TimescaleDB: TimescaleDBConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "postgres",
				Database: "cockpit_metrics",
				SSLMode:  "disable",
				MaxConns: 50,
				MinConns: 5,
			},
		},
		Cache: CacheConfig{
			Redis: RedisConfig{
				Addr:     "localhost:6379",
				PoolSize: 100,
			},
		},
		Auth: AuthConfig{
			JWTSecret:   "default-secret",
			TokenExpire: 3600,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}
