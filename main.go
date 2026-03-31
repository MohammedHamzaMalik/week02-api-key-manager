package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApiKey struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	KeyValue  string    `json:"keyValue"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateKeyRequest struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

var db *pgxpool.Pool

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func handleKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getAllKeys(w, r)
	case http.MethodPost:
		createKey(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleKeyByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		getKeyByID(w, r, id)
	case http.MethodDelete:
		deleteKey(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func getAllKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(context.Background(),
		"SELECT id, name, key_value, owner, created_at FROM api_keys ORDER BY id")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch keys")
		return
	}
	defer rows.Close()

	result := make([]ApiKey, 0)
	for rows.Next() {
		var k ApiKey
		err := rows.Scan(&k.ID, &k.Name, &k.KeyValue, &k.Owner, &k.CreatedAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan row")
			return
		}
		result = append(result, k)
	}
	writeJSON(w, http.StatusOK, result)
}

func createKey(w http.ResponseWriter, r *http.Request) {
	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name cannot be empty")
		return
	}
	if strings.TrimSpace(req.Owner) == "" {
		writeError(w, http.StatusBadRequest, "owner cannot be empty")
		return
	}

	// keyValue := fmt.Sprintf("%d-key-%d", time.Now().UnixNano(), time.Now().Unix())
	keyValue := strings.ReplaceAll(uuid.New().String(), "-", "")

	var key ApiKey
	err := db.QueryRow(context.Background(),
		`INSERT INTO api_keys (name, key_value, owner)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, key_value, owner, created_at`,
		req.Name, keyValue, req.Owner,
	).Scan(&key.ID, &key.Name, &key.KeyValue, &key.Owner, &key.CreatedAt)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create key")
		return
	}
	writeJSON(w, http.StatusCreated, key)
}

func getKeyByID(w http.ResponseWriter, r *http.Request, id int) {
	var key ApiKey
	err := db.QueryRow(context.Background(),
		"SELECT id, name, key_value, owner, created_at FROM api_keys WHERE id = $1",
		id,
	).Scan(&key.ID, &key.Name, &key.KeyValue, &key.Owner, &key.CreatedAt)

	if err != nil {
		writeError(w, http.StatusNotFound,
			fmt.Sprintf("key not found with id: %d", id))
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func deleteKey(w http.ResponseWriter, r *http.Request, id int) {
	result, err := db.Exec(context.Background(),
		"DELETE FROM api_keys WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete key")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound,
			fmt.Sprintf("key not found with id: %d", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	connStr := "postgres://postgres:root@localhost:5432/apikeymanager"
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("database ping failed:", err)
	}

	db = pool
	fmt.Println("Connected to PostgreSQL")
	fmt.Println("Server running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
