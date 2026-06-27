package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS imported (
		filename     TEXT    NOT NULL,
		size         INTEGER NOT NULL,
		imported_at  TEXT    NOT NULL,
		target_path  TEXT    NOT NULL,
		PRIMARY KEY (filename, size)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) IsImported(filename string, size int64) bool {
	var one int
	err := s.db.QueryRow("SELECT 1 FROM imported WHERE filename=? AND size=?", filename, size).Scan(&one)
	return err == nil
}

func (s *Store) Insert(filename string, size int64, importedAt, targetPath string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO imported (filename, size, imported_at, target_path) VALUES (?, ?, ?, ?)",
		filename, size, importedAt, targetPath,
	)
	if err != nil {
		return fmt.Errorf("insert imported: %w", err)
	}
	return nil
}
