package main

import (
	"log"
	"net/http"
	"time"
	"url-shortener/database"
	"url-shortener/handlers"
	"url-shortener/middleware"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	dbPool, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	redisClient, err := database.ConnectRDB()
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()

	urlHandler := &handlers.URLHandler{
		DB:  dbPool,
		RDB: redisClient,
	}

	rateLimiter := &middleware.RateLimitMiddleware{
		RDB: redisClient,
	}

	// 1. Build the Expensive Limiter (20 requests / 1 min / "shorten" prefix)
	expensiveMiddleware := rateLimiter.RateLimit(20, 1*time.Minute, "ratelimit:shorten")

	// 2. Build the Cheap Limiter (1000 requests / 1 min / "redirect" prefix)
	cheapMiddleware := rateLimiter.RateLimit(1000, 1*time.Minute, "ratelimit:redirect")

	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// Apply them to your routes!
	http.HandleFunc("/shorten", expensiveMiddleware(urlHandler.ShortenURLHandler))
	http.HandleFunc("/", cheapMiddleware(urlHandler.RedirectURLHandler))

	port := ":8080"
	log.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
