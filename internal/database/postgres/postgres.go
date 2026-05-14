package postgres

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"github.com/valpiks/backupctl/internal/config"
	database "github.com/valpiks/backupctl/internal/dbdriver"
)

var execCommandContext = exec.CommandContext

type Driver struct {
	cfg config.DatabaseConfig
}

func NewDriver(cfg config.DatabaseConfig) (*Driver, error) {
	if cfg.Type != "postgres" {
		return nil, fmt.Errorf("unsupported database type for postgres driver: %s", cfg.Type)
	}

	if cfg.Postgres.Host == "" {
		return nil, fmt.Errorf("postgres.host is required")
	}

	if cfg.Postgres.Port == 0 {
		return nil, fmt.Errorf("postgres.port is required")
	}

	if cfg.Postgres.User == "" {
		return nil, fmt.Errorf("postgres.user is required")
	}

	if cfg.Postgres.Name == "" {
		return nil, fmt.Errorf("postgres.name is required")
	}

	return &Driver{
		cfg: cfg,
	}, nil
}

func (d *Driver) Ping(ctx context.Context) error {
	args := []string{
		"-h", d.cfg.Postgres.Host,
		"-p", strconv.Itoa(d.cfg.Postgres.Port),
		"-U", d.cfg.Postgres.User,
		"-d", d.cfg.Postgres.Name,
		"-c", "select 1",
	}

	cmd := execCommandContext(ctx, "psql", args...)

	cmd.Env = append(cmd.Environ(), "PGPASSWORD="+d.cfg.Postgres.Password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postgres connection failed: %w: %s", err, string(output))
	}

	return nil
}

func (d *Driver) Backup(ctx context.Context, opts database.BackupOptions) (io.ReadCloser, error) {
	if opts.Type != "full" {
		return nil, fmt.Errorf("unsupported postgres backup type: %s", opts.Type)
	}

	if opts.ShemaOnly && opts.DataOnly {
		return nil, fmt.Errorf("schema-only and data-only cannot be used together")
	}

	if len(opts.Tables) > 0 && opts.ShemaOnly {
		return nil, fmt.Errorf("tables and schema-only cannot be combined safely")
	}

	var stderr bytes.Buffer

	args, err := buildBackupArgs(d.cfg, opts)
	if err != nil {
		return nil, err
	}

	cmd := execCommandContext(ctx, "pg_dump", args...)

	cmd.Env = append([]string{}, cmd.Environ()...)
	cmd.Env = append(cmd.Env, "PGPASSWORD="+d.cfg.Postgres.Password)

	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create pg_dump stdout pipe %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start pg_dump %w", err)
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
		targetDB = d.cfg.Postgres.Name
	}

	commandName, err := buildCommandNameForRestore(opts.Format)
	if err != nil {
		return err
	}

	args, err := buildRestoreArgs(d.cfg, targetDB, opts)
	if err != nil {
		return err
	}

	cmd := execCommandContext(ctx, commandName, args...)

	cmd.Env = append([]string{}, cmd.Environ()...)
	cmd.Env = append(cmd.Env, "PGPASSWORD="+d.cfg.Postgres.Password)

	cmd.Stdin = input

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postgres restore failed: %w: %s", err, string(output))
	}

	return nil
}

func buildBackupArgs(cfg config.DatabaseConfig, opts database.BackupOptions) ([]string, error) {
	args := []string{
		"-h", cfg.Postgres.Host,
		"-p", strconv.Itoa(cfg.Postgres.Port),
		"-U", cfg.Postgres.User,
		"-d", cfg.Postgres.Name,
		"--no-owner",
		"--no-privileges",
		"--no-password",
	}

	switch opts.Format {
	case "plain":
		args = append(args, "--format=plain")
	case "custom":
		args = append(args, "--format=custom", "-Fc")
	default:
		return nil, fmt.Errorf("unsupported postgres format: %s", opts.Format)
	}

	if opts.ShemaOnly {
		args = append(args, "--schema-only")
	}

	if opts.DataOnly {
		args = append(args, "--data-only")
	}

	for _, table := range opts.Tables {
		if table != "" {
			args = append(args, "--table", table)
		}
	}

	return args, nil
}

func buildRestoreArgs(cfg config.DatabaseConfig, targetDB string, opts database.RestoreOptions) ([]string, error) {
	args := []string{
		"-h", cfg.Postgres.Host,
		"-p", strconv.Itoa(cfg.Postgres.Port),
		"-U", cfg.Postgres.User,
		"-d", targetDB,
	}

	if opts.Format != "custom" && opts.Format != "plain" {
		return nil, fmt.Errorf("unsupported postgres format: %s", opts.Format)
	}

	if opts.Format == "custom" {
		args = append(args, "--clean", "--if-exists")
	}

	return args, nil
}

func buildCommandNameForRestore(format string) (string, error) {
	switch format {
	case "plain":
		return "psql", nil
	case "custom":
		return "pg_restore", nil
	default:
		return "", fmt.Errorf("unsupported postgres format: %s", format)
	}
}
