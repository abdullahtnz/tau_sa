package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func Connect() {
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	ctx := context.Background()

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatal("Failed to parse database config:", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnIdleTime = 1 * time.Minute

	DB, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("Failed to create connection pool:", err)
	}

	err = DB.Ping(ctx)
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to PostgreSQL database")

	go keepAlive()
}

func keepAlive() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err := DB.Ping(context.Background())
		if err != nil {
			log.Println("Keep-alive ping failed:", err)
		}
	}
}
