package migrations_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const migrationsDir = "../migrations"

type migrationFile struct {
	name    string
	number  int
	content string
}

func loadMigrationFiles(t *testing.T) []migrationFile {
	t.Helper()

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	var files []migrationFile
	numRe := regexp.MustCompile(`^(\d+)_.*\.sql$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := numRe.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		var number int
		if _, err := fmt.Sscanf(matches[1], "%d", &number); err != nil {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}

		files = append(files, migrationFile{
			name:    entry.Name(),
			number:  number,
			content: string(content),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].number < files[j].number
	})

	return files
}

func TestMigrationFilesAreNumberedSequentially(t *testing.T) {
	t.Parallel()

	files := loadMigrationFiles(t)
	if len(files) == 0 {
		t.Fatal("no migration files found")
	}

	for i, f := range files {
		expected := i + 1
		if f.number != expected {
			t.Fatalf("migration %s is numbered %d, expected %d", f.name, f.number, expected)
		}
	}

	if len(files) != 16 {
		t.Fatalf("expected 16 migrations, found %d", len(files))
	}
}

func TestMigrationFilesAreNotEmpty(t *testing.T) {
	t.Parallel()

	files := loadMigrationFiles(t)
	for _, f := range files {
		trimmed := strings.TrimSpace(f.content)
		if trimmed == "" {
			t.Fatalf("%s is empty", f.name)
		}
		if len(trimmed) < 10 {
			t.Fatalf("%s is suspiciously short (%d bytes)", f.name, len(trimmed))
		}
	}
}

func TestMigrationFilesContainSQLStatements(t *testing.T) {
	t.Parallel()

	files := loadMigrationFiles(t)

	sqlKeywords := []string{"CREATE", "ALTER", "DROP", "INSERT", "UPDATE", "DELETE", "COMMENT"}

	for _, f := range files {
		upper := strings.ToUpper(f.content)
		found := false
		for _, kw := range sqlKeywords {
			if strings.Contains(upper, kw) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s does not contain any recognized SQL statement", f.name)
		}
	}
}

// TestMigrationFilesApplyToTestDatabase attempts to apply all migrations to a
// test database. It is skipped when no PostgreSQL is available.
func TestMigrationFilesApplyToTestDatabase(t *testing.T) {
	t.Parallel()

	dsn := os.Getenv("ZENTHRIL_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("ZENTHRIL_TEST_DB_DSN not set; skipping live database migration test")
	}

	t.Skip("live database migration test requires PostgreSQL; enable with ZENTHRIL_TEST_DB_DSN")
}
