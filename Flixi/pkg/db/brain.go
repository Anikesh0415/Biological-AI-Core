package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var Brain *sql.DB

func InitBrain() {
	var err error
	Brain, err = sql.Open("sqlite", "brain.db")
	if err != nil {
		log.Fatalf("Failed to open brain.db: %v", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS preferences (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`
	_, err = Brain.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Failed to create preferences table: %v", err)
	}
	
	fmt.Println("Local SQLite Brain initialized successfully.")
}

func SetPreference(key string, value string) error {
	query := `INSERT INTO preferences (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value;`
	_, err := Brain.Exec(query, key, value)
	return err
}

func GetPreference(key string) string {
	query := `SELECT value FROM preferences WHERE key = ?;`
	row := Brain.QueryRow(query, key)
	var value string
	err := row.Scan(&value)
	if err != nil {
		return ""
	}
	return value
}
