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

type Driver struct {
	cfg config.DatabaseConfig
}

func NewDriver(cfg config.DatabaseConfig) (*Driver, error) {
	if cfg.Type != "postgres" {
		return nil, fmt.Errorf("unsupported database type for postgres driver: %s", cfg.Type)
	}

	if cfg.Host == "" {
		return nil, fmt.Errorf("database.host is required")
	}

	if cfg.Port == 0 {
		return nil, fmt.Errorf("database.port is required")
	}

	if cfg.User == "" {
		return nil, fmt.Errorf("database.user is required")
	}

	if cfg.Name == "" {
		return nil, fmt.Errorf("database.name is required")
	}

	return &Driver{
		cfg: cfg,
	}, nil
}

func (d *Driver) Ping(ctx context.Context) error {
	args := []string{
		"-h", d.cfg.Host,
		"-p", strconv.Itoa(d.cfg.Port),
		"-U", d.cfg.User,
		"-d", d.cfg.Name,
		"-c", "select 1",
	}

	cmd := exec.CommandContext(ctx, "psql", args...)

	cmd.Env = append(cmd.Environ(), "PGPASSWORD="+d.cfg.Password)

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
		"-h", d.cfg.Host,
		"-p", strconv.Itoa(d.cfg.Port),
		"-U", d.cfg.User,
		"-d", d.cfg.Name,
		"--format=plain",
		"--no-owner",
		"--no-privileges",
		"--no-password",
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)

	cmd.Env = append([]string{}, cmd.Environ()...)
	cmd.Env = append(cmd.Env, "PGPASSWORD="+d.cfg.Password)

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
		targetDB = d.cfg.Name
	}

	args := []string{
		"-h", d.cfg.Host,
		"-p", strconv.Itoa(d.cfg.Port),
		"-U", d.cfg.User,
		"-d", targetDB,
	}

	cmd := exec.CommandContext(ctx, "psql", args...)

	cmd.Env = append([]string{}, cmd.Environ()...)
	cmd.Env = append(cmd.Env, "PGPASSWORD="+d.cfg.Password)

	cmd.Stdin = input

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postgres restore failed: %w: %s", err, string(output))
	}

	return nil
}
