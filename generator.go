package migrations

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"
)

// GenerateOptions configures migration file generation.
type GenerateOptions struct {
	// Dir is the output directory. Defaults to "migrations".
	Dir string
	// PackageName is the Go package name. Defaults to the base name of Dir.
	PackageName string
	// Name is the human-readable migration name (e.g. "create_users_table"). Required.
	Name string
	// Timestamp overrides the version timestamp. Defaults to time.Now().
	Timestamp time.Time
}

var migrationTmpl = template.Must(template.New("migration").Parse(
	`package {{.PackageName}}

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
)

// {{.VarPrefix}}UpSQL and {{.VarPrefix}}DownSQL are package-level constants so that
// Checksum() automatically detects SQL body changes: edit the SQL here and the stored
// checksum will diverge, triggering ErrChecksumMismatch on the next run.
const {{.VarPrefix}}UpSQL = ` + "`" + `
	-- TODO: write your UP migration SQL here
` + "`" + `

const {{.VarPrefix}}DownSQL = ` + "`" + `
	-- TODO: write your DOWN migration SQL here
` + "`" + `

type {{.StructName}} struct{}

func (m {{.StructName}}) Version() string { return "{{.Version}}" }
func (m {{.StructName}}) Name() string    { return "{{.NameSnake}}" }

func (m {{.StructName}}) Up(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, {{.VarPrefix}}UpSQL)
	return err
}

func (m {{.StructName}}) Down(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, {{.VarPrefix}}DownSQL)
	return err
}

// Checksum implements ChecksumMigration. It is derived from the SQL body so that any
// edit to {{.VarPrefix}}UpSQL or {{.VarPrefix}}DownSQL is detected before the migration
// can be silently re-applied with a different schema.
func (m {{.StructName}}) Checksum() string {
	h := sha256.New()
	h.Write([]byte({{.VarPrefix}}UpSQL))
	h.Write([]byte{0})
	h.Write([]byte({{.VarPrefix}}DownSQL))
	return fmt.Sprintf("%x", h.Sum(nil))
}
`))

var listTmpl = template.Must(template.New("list").Parse(
	`package {{.PackageName}}

import base "github.com/akula410/migrations/v2"

// List contains all registered migrations in version order.
var List = []base.Migration{
{{- range .Structs}}
	{{.}}{},
{{- end}}
}
`))

// GenerateMigration writes a new migration file to opts.Dir.
func GenerateMigration(opts GenerateOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("generator: Name is required")
	}
	if opts.Dir == "" {
		opts.Dir = "migrations"
	}
	if opts.PackageName == "" {
		opts.PackageName = filepath.Base(opts.Dir)
	}
	if opts.Timestamp.IsZero() {
		opts.Timestamp = time.Now()
	}

	nameSnake := toSnakeCase(opts.Name)
	if nameSnake == "" {
		return fmt.Errorf("generator: invalid migration name %q", opts.Name)
	}

	version := opts.Timestamp.UTC().Format("20060102150405")
	structName := toPascalCase(nameSnake) + version

	data := struct {
		PackageName string
		StructName  string
		VarPrefix   string
		Version     string
		NameSnake   string
	}{
		PackageName: opts.PackageName,
		StructName:  structName,
		VarPrefix:   lowerFirst(structName),
		Version:     version,
		NameSnake:   nameSnake,
	}

	var buf bytes.Buffer
	if err := migrationTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("generator: template: %w", err)
	}

	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return fmt.Errorf("generator: mkdir: %w", err)
	}

	filename := filepath.Join(opts.Dir, version+"_"+nameSnake+".go")
	if err := os.WriteFile(filename, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("generator: write: %w", err)
	}
	return nil
}

// GenerateMigrationList writes (or overwrites) a list.go file that aggregates
// all struct names into a []base.Migration slice.
func GenerateMigrationList(dir, packageName string, structNames []string) error {
	if dir == "" {
		dir = "migrations"
	}
	if packageName == "" {
		packageName = filepath.Base(dir)
	}

	data := struct {
		PackageName string
		Structs     []string
	}{packageName, structNames}

	var buf bytes.Buffer
	if err := listTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("generator: list template: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("generator: mkdir: %w", err)
	}

	filename := filepath.Join(dir, "list.go")
	if err := os.WriteFile(filename, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("generator: write list: %w", err)
	}
	return nil
}

func toSnakeCase(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		} else if r == '_' {
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func toPascalCase(snake string) string {
	parts := strings.Split(snake, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	return b.String()
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}
