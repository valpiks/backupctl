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

	var stderr bytes.Buffer

	args := []string{
		"-h", d.cfg.Postgres.Host,
		"-p", strconv.Itoa(d.cfg.Postgres.Port),
		"-U", d.cfg.Postgres.User,
		"-d", d.cfg.Postgres.Name,
		"--format=plain",
		"--no-owner",
		"--no-privileges",
		"--no-password",
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

	args := []string{
		"-h", d.cfg.Postgres.Host,
		"-p", strconv.Itoa(d.cfg.Postgres.Port),
		"-U", d.cfg.Postgres.User,
		"-d", targetDB,
	}

	cmd := execCommandContext(ctx, "psql", args...)

	cmd.Env = append([]string{}, cmd.Environ()...)
	cmd.Env = append(cmd.Env, "PGPASSWORD="+d.cfg.Postgres.Password)

	cmd.Stdin = input

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postgres restore failed: %w: %s", err, string(output))
	}

	return nil
}
