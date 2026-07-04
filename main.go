package main

import (
	"ego/database"
	"ego/handlers"
	"ego/templates"
	"log"
	"net/http"
	"os"
)

func main() {
	// Initialize database
	database.Init()

	// Initialize templates
	templates.Init()

	// Setup routes
	router := handlers.Router()

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Hikvision Broadcast Management System starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
