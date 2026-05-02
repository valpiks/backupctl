package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Backup   BackupConfig   `yaml:"backup"`
	Storage  StorageConfig  `yaml:"storage"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type DatabaseConfig struct {
	Type     string         `yaml:"type"`
	Postgres PostgresConfig `yaml:"postgres"`
	Mongo    MongoConfig    `yaml:"mongo"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
}

type MongoConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type BackupConfig struct {
	Type        string `yaml:"type"`
	Compression string `yaml:"compression"`
}

type StorageConfig struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %w", err)
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Database.Type == "" {
		return fmt.Errorf("database.type is required")
	}

	if c.Database.Type == "postgres" && c.Database.Postgres.Name == "" {
		return fmt.Errorf("postgres.name is required")
	}

	if c.Database.Type == "mongo" && c.Database.Mongo.Database == "" {
		return fmt.Errorf("mongo.database is required")
	}

	if c.Backup.Type == "" {
		return fmt.Errorf("backup.type is required")
	}

	if c.Storage.Type == "" {
		return fmt.Errorf("storage.type is required")
	}

	if c.Storage.Path == "" {
		return fmt.Errorf("storage.path is required")
	}

	return nil
}

func (d DatabaseConfig) ActiveDatabaseName() string {
	switch d.Type {
	case "postgres":
		return d.Postgres.Name
	case "mongo":
		return d.Mongo.Database
	default:
		return ""
	}
}
