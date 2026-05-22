package database

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to PostgreSQL database")
}
