package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type user struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func userHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Your existing POST logic goes here.

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return

		}
		var newUser user
		err := json.NewDecoder(r.Body).Decode(&newUser)
		if err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(newUser.Name) == "" ||
			strings.TrimSpace(newUser.Email) == "" {
			http.Error(w, "name and email are required", http.StatusBadRequest)
			return
		}

		query := `
            INSERT INTO users (name, email)
            VALUES ($1, $2)
            RETURNING id, created_at
        `

		err = db.QueryRowContext(
			r.Context(),
			query,
			newUser.Name,
			newUser.Email,
		).Scan(&newUser.ID, &newUser.CreatedAt)

		if err != nil {
			log.Printf("failed to create user: %v", err)
			http.Error(w, "could not create user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(newUser); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	}
}
func listUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rows, err := db.QueryContext(
			r.Context(),
			"SELECT id, name, email, created_at FROM users ORDER BY id",
		)

		if err != nil {
			log.Printf("failed to fetch users: %v", err)
			http.Error(w, "could not fetch users", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		usersList := make([]user, 0)

		for rows.Next() {
			var u user

			err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
			if err != nil {
				log.Printf("failed to scan user: %v", err)
				http.Error(w, "could not read users", http.StatusInternalServerError)
				return
			}

			usersList = append(usersList, u)
		}

		if err := rows.Err(); err != nil {
			log.Printf("error iterating users: %v", err)
			http.Error(w, "could not read users", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(usersList); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	}
}

type healthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := healthResponse{
		Status:  "ok",
		Message: "Task Management API is running",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func connectDB() (*sql.DB, error) {
	// Read the database URL from the environment.
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	// Create a database handle using the pgx driver.
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify that PostgreSQL is actually reachable.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return db, nil
}

func main() {

	db, err := connectDB()

	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	defer db.Close()

	log.Println("Successfully connected to PostgreSQL")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /users", userHandler(db))
	mux.HandleFunc("GET /users", listUsersHandler(db))

	addr := ":8081"

	log.Printf("Server listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
