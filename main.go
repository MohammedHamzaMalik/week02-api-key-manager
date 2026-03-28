package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type APIKey struct {
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

// In-memory store - no database yet
var (
	keys   = make(map[int]APIKey)
	nextID = 1
	mu     sync.Mutex
)

func generateKey() string {
	return fmt.Sprintf("%d-key-%d", time.Now().UnixNano(), nextID)
}

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
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleKeyByID(w http.ResponseWriter, r *http.Request) {
	// extract ID from path: /api/keys/123
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
		deleteKeyByID(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func getAllKeys(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	result := make([]APIKey, 0, len(keys))
	for _, key := range keys {
		result = append(result, key)
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
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if strings.TrimSpace(req.Owner) == "" {
		writeError(w, http.StatusBadRequest, "owner is required")
		return
	}

	mu.Lock()
	defer mu.Unlock()
	key := APIKey{
		ID:        nextID,
		Name:      req.Name,
		KeyValue:  generateKey(),
		Owner:     req.Owner,
		CreatedAt: time.Now(),
	}
	keys[nextID] = key
	nextID++

	writeJSON(w, http.StatusCreated, key)
}

func getKeyByID(w http.ResponseWriter, r *http.Request, id int) {
	mu.Lock()
	defer mu.Unlock()

	key, exists := keys[id]
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("key not found with id: %d", id))
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func deleteKeyByID(w http.ResponseWriter, r *http.Request, id int) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := keys[id]; !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("key not found with id: %d", id))
		return
	}
	delete(keys, id)
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	http.HandleFunc("/api/keys", handleKeys)
	http.HandleFunc("/api/keys/", handleKeyByID)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
