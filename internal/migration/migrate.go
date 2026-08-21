package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Runner struct {
	pool *pgxpool.Pool
	dir  string
}

func NewRunner(pool *pgxpool.Pool, dir string) *Runner {
	return &Runner{pool: pool, dir: dir}
}

func (r *Runner) Run(ctx context.Context) error {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return fmt.Errorf("read migration dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(r.dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := r.pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
