package main

import (
	"authenticaiton/data"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	webServerPort = "80"
)

var counts int64

type Config struct {
	DB     *sql.DB
	Models data.Models
}

func connectDB() *sql.DB {
	dsn := os.Getenv("DSN")
	var counts int

	for {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Printf("Error opening database: %v", err)
			counts++
		} else if err := db.Ping(); err != nil {
			log.Printf("Error pinging database: %v", err)
			counts++
			_ = db.Close() // Close DB on failure
		} else {
			log.Println("Connected to the database")
			return db
		}

		if counts > 20 {
			log.Fatalf("Failed to connect to the database after %d tries. Last error: %v", counts, err)
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

func main() {
	log.Println("Starting the authentication service")

	// Routes.

	// Connect to the database.
	db := connectDB()
	if db == nil {
		log.Fatalf("failed to connect to the database")
	}

	// Set up configuration.
	app := Config{
		DB:     db,
		Models: data.New(db),
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webServerPort),
		Handler: app.routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
	log.Println("Server started")

}
