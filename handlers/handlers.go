package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type URLHandler struct {
	DB *pgxpool.Pool
	// Tip: It's often safer to inject a BaseURL string here
	// (e.g., "https://sho.rt") rather than relying on r.Host.
	// BaseURL string
	RDB *redis.Client
}

type ShortenURLRequest struct {
	URL string `json:"url"`
}

const s = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func Base62(n int) string {
	if n == 0 {
		return "0"
	}

	var result []byte

	for n > 0 {
		result = append(result, s[n%62])
		n /= 62
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

func (h *URLHandler) ShortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	query := `INSERT INTO urls (url) VALUES ($1) RETURNING id`

	var id int
	err := h.DB.QueryRow(r.Context(), query, req.URL).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
		return
	}

	short_code := Base62(id)

	query = `UPDATE urls SET short_code = $1 WHERE id = $2`
	res, err := h.DB.Exec(r.Context(), query, short_code, id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if res.RowsAffected() == 0 {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Determine scheme (http vs https)
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	// Construct the final short URL using the request host.
	// If you added BaseURL to URLHandler, you would use:
	// short_url := fmt.Sprintf("%s/%s", h.BaseURL, short_code)
	short_url := fmt.Sprintf("%s://%s/%s", scheme, r.Host, short_code)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Complete the JSON encoding block
	json.NewEncoder(w).Encode(map[string]string{
		"short_url": short_url,
	})
}

func (h *URLHandler) RedirectURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortCode := strings.TrimPrefix(r.URL.Path, "/")

	if shortCode == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	longURL, err := h.RDB.Get(r.Context(), shortCode).Result()
	if err == redis.Nil {
		query := `SELECT url FROM urls WHERE short_code = $1`
		err := h.DB.QueryRow(r.Context(), query, shortCode).Scan(&longURL)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "Short code not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		h.RDB.Set(r.Context(), shortCode, longURL, 24*time.Hour)
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusPermanentRedirect)
}
