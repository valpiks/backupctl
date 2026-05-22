package config

import (
	"fmt"
	"os"
	"time"

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

	Scheduler *SchedulerConfig `yaml:"scheduler,omitempty"`
}

type StorageConfig struct {
	Type  string             `yaml:"type"`
	Local LocalStorageConfig `yaml:"local"`
	S3    S3StorageConfig    `yaml:"s3"`
}

type LocalStorageConfig struct {
	Path string `yaml:"path"`
}

type S3StorageConfig struct {
	Bucket         string `yaml:"bucket"`
	Region         string `yaml:"region"`
	Prefix         string `yaml:"prefix"`
	Endpoint       string `yaml:"endpoint"`
	ForcePathStyle bool   `yaml:"force_path_style"`
}

type SchedulerConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Cron     string `yaml:"cron,omitempty"`
	Interval string `yaml:"interval,omitempty"`
	LogFile  string `yaml:"log_file,omitempty"`
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

	scheduler := c.Backup.Scheduler
	if scheduler != nil && scheduler.Enabled {
		if scheduler.Interval == "" && scheduler.Cron == "" {
			return fmt.Errorf("either scheduler.interval or scheduler.cron is required")
		}

		if scheduler.Interval != "" && scheduler.Cron != "" {
			return fmt.Errorf("scheduler.interval and scheduler.cron cannot be used together")
		}
	}

	if scheduler != nil && scheduler.Interval != "" {
		if _, err := time.ParseDuration(scheduler.Interval); err != nil {
			return fmt.Errorf("invalid scheduler.interval: %w", err)
		}
	}

	if c.Storage.Type == "" {
		return fmt.Errorf("storage.type is required")
	}

	switch c.Storage.Type {
	case "local":
		if c.Storage.Local.Path == "" {
			return fmt.Errorf("storage.local.path is required for local storage")
		}
	case "s3":
		if c.Storage.S3.Bucket == "" {
			return fmt.Errorf("s3.bucket is required")
		}
		if c.Storage.S3.Region == "" {
			return fmt.Errorf("s3.region is required (use 'us-east-1' for MinIO, 'auto' for R2)")
		}
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
