package main

import (
	"log"
	"net/http"
	"url-shortener/database"
	"url-shortener/handlers"
)

func main() {
	dbPool, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	urlHandler := handlers.URLHandler{
		DB: dbPool,
	}

	http.HandleFunc("/shorten", urlHandler.ShortenURLHandler)

	http.HandleFunc("/", urlHandler.RedirectURLHandler)

	port := ":8080"
	log.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
