package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
)

type user struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users []user
var usersMu sync.RWMutex

func userHandler(w http.ResponseWriter, r *http.Request) {
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

	usersMu.Lock()
	users = append(users, newUser)
	usersMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(newUser); err != nil {
		log.Printf("failed to encode response: %v", err)
	}

}
func listUsersHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return

	}

	usersMu.RLock()
	usersList := make([]user, len(users))
	copy(usersList, users)
	usersMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(usersList); err != nil {
		log.Printf("failed to encode response: %v", err)
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

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /users", userHandler)
	mux.HandleFunc("GET /users", listUsersHandler)

	addr := ":8081"

	log.Printf("Server listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
