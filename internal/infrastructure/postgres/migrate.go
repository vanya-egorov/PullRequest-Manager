package postgres

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	var files []fs.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, entry)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	for _, file := range files {
		version := strings.SplitN(file.Name(), "_", 2)[0]
		err = pool.QueryRow(ctx, `SELECT true FROM schema_migrations WHERE version=$1`, version).Scan(new(bool))
		if err == nil {
			continue
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		path := filepath.Join(migrationsDir, file.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if _, execErr := pool.Exec(ctx, string(data)); execErr != nil {
			return execErr
		}
		_, insertErr := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, version)
		if insertErr != nil {
			return insertErr
		}
	}
	return nil
}
