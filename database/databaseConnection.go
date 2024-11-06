package database

import (
    "database/sql"
    "fmt"
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
            // Try to ping the database
            err = db.Ping()
            if err == nil {
                fmt.Println("Successfully connected to the database!")
                return db, nil
            }
        }

        // If an error occurs, log the attempt and wait before retrying
        fmt.Printf("Attempt %d: Failed to connect to the database. Retrying in 2 seconds...\n", attempts)
        time.Sleep(2 * time.Second)
    }

    // After 5 attempts, return the error
    return nil, fmt.Errorf("failed to connect to the database after 5 attempts: %w", err)
}