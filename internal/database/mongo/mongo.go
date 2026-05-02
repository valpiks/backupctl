package mongo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/valpiks/backupctl/internal/config"
	database "github.com/valpiks/backupctl/internal/dbdriver"
)

var execCommandContext = exec.CommandContext

type Driver struct {
	cfg config.DatabaseConfig
}

func NewDriver(cfg config.DatabaseConfig) (*Driver, error) {
	if cfg.Type != "mongo" {
		return nil, fmt.Errorf("unsupported database type for mongo driver: %s", cfg.Type)
	}

	if cfg.Mongo.URI == "" {
		return nil, fmt.Errorf("mongo.uri is required")
	}

	if cfg.Mongo.Database == "" {
		return nil, fmt.Errorf("mongo.database is required")
	}

	return &Driver{
		cfg: cfg,
	}, nil
}

func (d *Driver) Ping(ctx context.Context) error {
	args := []string{
		d.cfg.Mongo.URI,
		"--eval", "db.adminCommand('ping')",
	}

	cmd := execCommandContext(ctx, "mongosh", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mongo connection failed: %w: %s", err, string(output))
	}

	return nil
}

func (d *Driver) Backup(ctx context.Context, opts database.BackupOptions) (io.ReadCloser, error) {
	if opts.Type != "full" {
		return nil, fmt.Errorf("unsupported mongo backup type: %s", opts.Type)
	}

	var stderr bytes.Buffer

	args := []string{
		"--uri", d.cfg.Mongo.URI,
		"--db", d.cfg.Mongo.Database,
		"--archive",
	}

	cmd := execCommandContext(ctx, "mongodump", args...)

	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create mongodump stdout pipe %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mongodump %w", err)
	}

	return &commandReadCloser{
		Reader: stdout,
		cmd:    cmd,
		stderr: &stderr,
	}, nil
}

type commandReadCloser struct {
	io.Reader
	cmd    *exec.Cmd
	stderr *bytes.Buffer
}

func (c *commandReadCloser) Close() error {
	if err := c.cmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %w, %s", err, c.stderr.String())
	}

	return nil
}

func (d *Driver) Restore(ctx context.Context, input io.Reader, opts database.RestoreOptions) error {
	targetDB := opts.TargetDatabase
	if targetDB == "" {
		targetDB = d.cfg.Mongo.Database
	}

	args := []string{
		"--uri", d.cfg.Mongo.URI,
		"--db", targetDB,
		"--archive=-",
	}

	cmd := execCommandContext(ctx, "mongorestore", args...)

	cmd.Stdin = input

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mongo restore failed: %w: %s", err, string(output))
	}

	return nil
}
