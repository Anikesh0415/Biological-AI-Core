package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"

	_ "modernc.org/sqlite"
)

type Action struct {
	Type string `json:"type"`
	Key  string `json:"key,omitempty"`
	Text string `json:"text,omitempty"`
	Name string `json:"name,omitempty"`
	Ms   int    `json:"ms,omitempty"`
}

func main() {
	fmt.Println("Starting 8-Million App Database Generator...")

	os.Remove("app_actions.db") // Start fresh

	db, err := sql.Open("sqlite", "app_actions.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS macros (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			intent TEXT NOT NULL,
			actions TEXT NOT NULL
		);
		CREATE INDEX idx_intent ON macros(intent);
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Use transactions for massive speedup
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("INSERT INTO macros(intent, actions) VALUES(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	actionsList := [][]Action{
		{
			{Type: "key", Key: "win+r"},
			{Type: "sleep", Ms: 500},
			{Type: "type", Text: "explorer"},
			{Type: "key", Key: "enter"},
		},
		{
			{Type: "key", Key: "ctrl+s"},
			{Type: "sleep", Ms: 300},
			{Type: "type", Text: "document"},
			{Type: "key", Key: "enter"},
		},
		{
			{Type: "key", Key: "alt+f4"},
		},
	}

	const totalRecords = 80000000
	const batchSize = 100000

	for i := 1; i <= totalRecords; i++ {
		intent := fmt.Sprintf("do action %d on app_%d", rand.Intn(100), i)
		actionData, _ := json.Marshal(actionsList[i%len(actionsList)])

		_, err = stmt.Exec(intent, string(actionData))
		if err != nil {
			log.Fatal(err)
		}

		if i%batchSize == 0 {
			fmt.Printf("Generated %d / %d apps...\n", i, totalRecords)
			err = tx.Commit()
			if err != nil {
				log.Fatal(err)
			}
			tx, _ = db.Begin()
			stmt, _ = tx.Prepare("INSERT INTO macros(intent, actions) VALUES(?, ?)")
		}
	}
	tx.Commit()

	fmt.Println("Successfully generated 80,000,000 application macros in app_actions.db!")
}
