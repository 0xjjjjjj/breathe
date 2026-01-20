package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type OpType string

const (
	OpMove   OpType = "move"
	OpDelete OpType = "delete"
	OpTrash  OpType = "trash"
)

type Operation struct {
	ID         int64
	Timestamp  time.Time
	Type       OpType
	SourcePath string
	DestPath   string
	FileSize   int64
	FileHash   string
	Reversible bool
	Metadata   map[string]string
}

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS operations (
			id INTEGER PRIMARY KEY,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			operation TEXT NOT NULL,
			source_path TEXT NOT NULL,
			dest_path TEXT,
			file_size INTEGER,
			file_hash TEXT,
			reversible BOOLEAN DEFAULT 0,
			metadata JSON
		);
		CREATE INDEX IF NOT EXISTS idx_source ON operations(source_path);
		CREATE INDEX IF NOT EXISTS idx_timestamp ON operations(timestamp);
	`)
	return err
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) Record(op Operation) (int64, error) {
	metadata, _ := json.Marshal(op.Metadata)

	result, err := d.db.Exec(`
		INSERT INTO operations (operation, source_path, dest_path, file_size, file_hash, reversible, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, op.Type, op.SourcePath, op.DestPath, op.FileSize, op.FileHash, op.Reversible, string(metadata))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DB) Search(query string) ([]Operation, error) {
	pattern := "%" + query + "%"
	rows, err := d.db.Query(`
		SELECT id, timestamp, operation, source_path, dest_path, file_size, file_hash, reversible, metadata
		FROM operations
		WHERE source_path LIKE ? OR dest_path LIKE ?
		ORDER BY timestamp DESC
	`, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOperations(rows)
}

func (d *DB) Since(t time.Time) ([]Operation, error) {
	rows, err := d.db.Query(`
		SELECT id, timestamp, operation, source_path, dest_path, file_size, file_hash, reversible, metadata
		FROM operations
		WHERE timestamp >= ?
		ORDER BY timestamp DESC
	`, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOperations(rows)
}

func (d *DB) Get(id int64) (*Operation, error) {
	row := d.db.QueryRow(`
		SELECT id, timestamp, operation, source_path, dest_path, file_size, file_hash, reversible, metadata
		FROM operations WHERE id = ?
	`, id)

	var op Operation
	var ts string
	var destPath sql.NullString
	var metadata sql.NullString

	err := row.Scan(&op.ID, &ts, &op.Type, &op.SourcePath, &destPath, &op.FileSize, &op.FileHash, &op.Reversible, &metadata)
	if err != nil {
		return nil, err
	}

	op.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
	if destPath.Valid {
		op.DestPath = destPath.String
	}
	if metadata.Valid {
		json.Unmarshal([]byte(metadata.String), &op.Metadata)
	}

	return &op, nil
}

func scanOperations(rows *sql.Rows) ([]Operation, error) {
	var ops []Operation
	for rows.Next() {
		var op Operation
		var ts string
		var destPath sql.NullString
		var metadata sql.NullString

		err := rows.Scan(&op.ID, &ts, &op.Type, &op.SourcePath, &destPath, &op.FileSize, &op.FileHash, &op.Reversible, &metadata)
		if err != nil {
			return nil, err
		}

		op.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		if destPath.Valid {
			op.DestPath = destPath.String
		}
		if metadata.Valid {
			json.Unmarshal([]byte(metadata.String), &op.Metadata)
		}

		ops = append(ops, op)
	}
	return ops, rows.Err()
}

func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
