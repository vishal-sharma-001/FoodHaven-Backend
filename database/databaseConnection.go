package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func ConnectDB() (*sql.DB, error) {
	host     := os.Getenv("DB_HOST")
	port     := os.Getenv("DB_PORT")
	user     := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname   := os.Getenv("DB_NAME")

	log.Printf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	var db *sql.DB
	var err error
	for attempts := 1; attempts <= 5; attempts++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				return db, nil
			}
		}

		log.Printf("Attempt %d: Failed to connect to the database. Retrying in 2 seconds...\n", attempts)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect to the database after 5 attempts: %w", err)
}
