package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	migrations "github.com/akula410/migrations"
)

func TestGenerateMigration_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	if err := migrations.GenerateMigration(migrations.GenerateOptions{
		Dir:         dir,
		PackageName: "mymigrations",
		Name:        "create_users_table",
		Timestamp:   ts,
	}); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}

	expected := filepath.Join(dir, "20260608120000_create_users_table.go")
	data, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "CreateUsersTable20260608120000") {
		t.Errorf("struct name not found in: %s", content)
	}
	if !strings.Contains(content, `"20260608120000"`) {
		t.Errorf("version not found in: %s", content)
	}
	if !strings.Contains(content, `"create_users_table"`) {
		t.Errorf("name not found in: %s", content)
	}
	if !strings.Contains(content, "package mymigrations") {
		t.Errorf("package name not found in: %s", content)
	}
}

func TestGenerateMigration_RequiresName(t *testing.T) {
	dir := t.TempDir()
	err := migrations.GenerateMigration(migrations.GenerateOptions{Dir: dir})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestGenerateMigration_DefaultsDir(t *testing.T) {
	// Use a temp dir as working directory
	orig, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Skip("cannot chdir")
	}
	defer os.Chdir(orig)

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := migrations.GenerateMigration(migrations.GenerateOptions{
		Name:      "init_schema",
		Timestamp: ts,
	}); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}

	_, err := os.Stat(filepath.Join(tmp, "migrations", "20260101000000_init_schema.go"))
	if err != nil {
		t.Fatalf("expected file in default dir: %v", err)
	}
}

func TestGenerateMigrationList(t *testing.T) {
	dir := t.TempDir()
	structs := []string{"CreateUsersTable20260608120000", "AddEmailIndex20260609000000"}

	if err := migrations.GenerateMigrationList(dir, "mymigrations", structs); err != nil {
		t.Fatalf("GenerateMigrationList: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "list.go"))
	if err != nil {
		t.Fatalf("list.go not created: %v", err)
	}

	content := string(data)
	for _, s := range structs {
		if !strings.Contains(content, s+"{}") {
			t.Errorf("struct %s not found in list.go", s)
		}
	}
	if !strings.Contains(content, "package mymigrations") {
		t.Error("package name not found in list.go")
	}
}

func TestGenerateMigration_NormalizesName(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Spaces and dashes should be normalised to underscores
	if err := migrations.GenerateMigration(migrations.GenerateOptions{
		Dir:       dir,
		Name:      "add email index",
		Timestamp: ts,
	}); err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}

	expected := filepath.Join(dir, "20260101000000_add_email_index.go")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected normalised filename: %v", err)
	}
}

func TestGenerateMigration_InvalidName(t *testing.T) {
	dir := t.TempDir()
	err := migrations.GenerateMigration(migrations.GenerateOptions{
		Dir:  dir,
		Name: "!!!",
	})
	if err == nil {
		t.Fatal("expected error for all-special-char name")
	}
}
