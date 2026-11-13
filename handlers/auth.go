package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"todo-journal/database"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// RegisterHandler handles new user sign-up
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Name == "" {
		JSONError(w, "Full name is required", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		JSONError(w, "Email address is required", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		JSONError(w, "Invalid email address format", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		JSONError(w, "Password must be at least 6 characters long", http.StatusBadRequest)
		return
	}

	// Check if email already registered
	var existingID string
	err := database.DB.QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&existingID)
	if err == nil {
		JSONError(w, "An account with this email address already exists", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows {
		JSONError(w, "Database error checking email", http.StatusInternalServerError)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		JSONError(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	userID := uuid.New().String()
	now := time.Now()

	_, err = database.DB.Exec(
		"INSERT INTO users (id, name, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)",
		userID, req.Name, req.Email, string(hashedPassword), now,
	)
	if err != nil {
		JSONError(w, "Failed to create user account", http.StatusInternalServerError)
		return
	}

	token, err := GenerateToken(userID, req.Email, req.Name)
	if err != nil {
		JSONError(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	// Also add welcome to-dos for new user
	seedWelcomeTodos(userID)

	JSONResponse(w, http.StatusCreated, AuthResponse{
		Token: token,
		User: UserResponse{
			ID:        userID,
			Name:      req.Name,
			Email:     req.Email,
			CreatedAt: now,
		},
	})
}

// LoginHandler handles user sign-in
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || req.Password == "" {
		JSONError(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	var user UserResponse
	var passwordHash string
	var createdAtStr string

	err := database.DB.QueryRow(
		"SELECT id, name, email, password_hash, created_at FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &passwordHash, &createdAtStr)

	if err == sql.ErrNoRows {
		JSONError(w, "Invalid email or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		JSONError(w, "Database query error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		JSONError(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	parsedCreatedAt, _ := time.Parse("2006-01-02 15:04:05", createdAtStr)
	user.CreatedAt = parsedCreatedAt

	token, err := GenerateToken(user.ID, user.Email, user.Name)
	if err != nil {
		JSONError(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	JSONResponse(w, http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

// MeHandler returns the authenticated user's profile
func MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(UserContextKey).(*TokenClaims)
	if !ok || claims == nil {
		JSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user UserResponse
	var createdAtStr string
	err := database.DB.QueryRow(
		"SELECT id, name, email, created_at FROM users WHERE id = ?",
		claims.UserID,
	).Scan(&user.ID, &user.Name, &user.Email, &createdAtStr)

	if err != nil {
		JSONError(w, "User not found", http.StatusNotFound)
		return
	}

	user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	JSONResponse(w, http.StatusOK, map[string]interface{}{"user": user})
}

// SeedDemoUser creates a pre-populated demo account if none exists
func SeedDemoUser() {
	var count int
	_ = database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = 'demo@example.com'").Scan(&count)
	if count > 0 {
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	demoID := "demo-user-0001"
	_, err := database.DB.Exec(
		"INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		demoID, "Demo Explorer", "demo@example.com", string(hashedPassword),
	)
	if err == nil {
		seedWelcomeTodos(demoID)
	}
}

func seedWelcomeTodos(userID string) {
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour).Format("2006-01-02 14:00")
	nextWeek := now.Add(7 * 24 * time.Hour).Format("2006-01-02 09:30")
	completedTime := now.Add(-2 * time.Hour).Format("2006-01-02 15:04:05")

	todos := []struct {
		Title       string
		Completed   int
		Priority    string
		ReminderAt  *string
		CompletedAt *string
	}{
		{"Welcome to Personal Task Manager! 🚀", 0, "high", &tomorrow, nil},
		{"Set reminders with custom date & time when creating or editing tasks", 0, "medium", &nextWeek, nil},
		{"Filter tasks by Priority (Low/Med/High) and Date (Today/Upcoming/Overdue)", 0, "high", nil, nil},
		{"Try editing this task inline by clicking the pencil or double-clicking", 0, "medium", nil, nil},
		{"Mark this item complete to see it smoothly sink to the bottom with completion timestamp", 0, "low", nil, nil},
		{"This completed task has a saved completion date and time", 1, "low", nil, &completedTime},
	}

	for _, t := range todos {
		id := uuid.New().String()
		nowStr := now.Format("2006-01-02 15:04:05")
		_, _ = database.DB.Exec(
			"INSERT INTO todos (id, user_id, title, completed, priority, reminder_at, completed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, userID, t.Title, t.Completed, t.Priority, t.ReminderAt, t.CompletedAt, nowStr, nowStr,
		)
	}
}
