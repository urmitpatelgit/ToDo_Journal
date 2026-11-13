package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"todo-journal/database"
	"todo-journal/handlers"
)

func main() {
	// 1. Initialize SQLite Database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "todos.db"
	}

	_, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}
	defer database.DB.Close()

	// Seed demo user for immediate evaluation
	handlers.SeedDemoUser()

	// 2. Setup HTTP Router
	mux := http.NewServeMux()

	// Auth Endpoints
	mux.HandleFunc("/api/auth/register", handlers.RegisterHandler)
	mux.HandleFunc("/api/auth/login", handlers.LoginHandler)
	mux.HandleFunc("/api/auth/me", handlers.AuthMiddleware(handlers.MeHandler))

	// Todo Endpoints
	mux.HandleFunc("/api/todos", handlers.AuthMiddleware(handlers.TodosHandler))
	mux.HandleFunc("/api/todos/", handlers.AuthMiddleware(handlers.TodoItemHandler))

	// Health check endpoint
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSONResponse(w, http.StatusOK, map[string]string{
			"status": "healthy",
			"db":     "connected",
		})
	})

	// Static Files Frontend (React + Tailwind CSS)
	fs := http.FileServer(http.Dir("./static"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Prevent directory traversal or missing routes from falling through to API
		if strings.HasPrefix(r.URL.Path, "/api/") {
			handlers.JSONError(w, "Endpoint not found", http.StatusNotFound)
			return
		}
		fs.ServeHTTP(w, r)
	})

	// Wrap entire mux with CORS handler
	handler := handlers.CORSMiddleware(mux)

	// Determine Port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf("0.0.0.0:%s", port)

	fmt.Println("==================================================")
	fmt.Println("  🚀 Personal Task Manager Server (Go + SQLite) ")
	fmt.Println("==================================================")
	fmt.Printf("  • Local URL:            http://localhost:%s\n", port)
	fmt.Printf("  • Network / Public:     http://0.0.0.0:%s\n", port)
	fmt.Printf("  • SQLite Database:      %s\n", dbPath)
	fmt.Println("  • Demo Account:         demo@example.com / demo123")
	fmt.Println("==================================================")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
