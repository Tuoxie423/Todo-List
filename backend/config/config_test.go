package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromPath(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir)

	cfg, err := LoadFromPath(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.Port != "18080" {
		t.Fatalf("expected backend port 18080, got %q", cfg.Server.Port)
	}
	if cfg.Server.Mode != "debug" {
		t.Fatalf("expected backend mode debug, got %q", cfg.Server.Mode)
	}
	if cfg.DB.DBName != "todo_list_test" {
		t.Fatalf("expected db name todo_list_test, got %q", cfg.DB.DBName)
	}
}

func TestLoadUsesConfigPathEnv(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir)
	t.Setenv("CONFIG_PATH", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DB.DSN() != "root:secret@tcp(localhost:3306)/todo_list_test?charset=utf8mb4&parseTime=True&loc=Local" {
		t.Fatalf("unexpected dsn: %s", cfg.DB.DSN())
	}
}

func TestLoadUsesConfigPathEnvFile(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir)
	t.Setenv("CONFIG_PATH", filepath.Join(dir, "config.yaml"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.Port != "18080" {
		t.Fatalf("expected backend port 18080, got %q", cfg.Server.Port)
	}
	if cfg.DB.DBName != "todo_list_test" {
		t.Fatalf("expected db name todo_list_test, got %q", cfg.DB.DBName)
	}
}

func TestLoadFromPathReturnsError(t *testing.T) {
	_, err := LoadFromPath(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
	if !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("expected read config error, got %v", err)
	}
}

func writeTestConfig(t *testing.T, dir string) {
	t.Helper()

	content := []byte(`backend:
  port: 18080
  mode: debug

database:
  host: localhost
  port: 3306
  user: root
  password: secret
  name: todo_list_test
  charset: utf8mb4
`)

	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), content, 0644); err != nil {
		t.Fatalf("write test config: %v", err)
	}
}
