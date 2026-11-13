package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB initializes SQLite database connection and runs schema migrations
func InitDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		dbPath = "todos.db"
	}

	// Ensure directory exists if path has subdirectories
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	// Open database with WAL mode and foreign keys enabled
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	log.Printf("SQLite database initialized successfully at: %s", dbPath)
	return db, nil
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL COLLATE NOCASE,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS todos (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		completed INTEGER DEFAULT 0,
		priority TEXT DEFAULT 'medium',
		reminder_at TEXT,
		completed_at TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id);
	CREATE INDEX IF NOT EXISTS idx_todos_user_completed ON todos(user_id, completed, created_at);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Safe column migration for existing databases
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name = 'reminder_at'").Scan(&count)
	if count == 0 {
		_, _ = db.Exec("ALTER TABLE todos ADD COLUMN reminder_at TEXT")
	}

	count = 0
	_ = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name = 'completed_at'").Scan(&count)
	if count == 0 {
		_, _ = db.Exec("ALTER TABLE todos ADD COLUMN completed_at TEXT")
	}

	return nil
}
