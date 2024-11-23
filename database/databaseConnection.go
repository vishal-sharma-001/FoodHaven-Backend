package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "Visz7637@"
	dbname   = "godb"
)

func ConnectDB() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

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
