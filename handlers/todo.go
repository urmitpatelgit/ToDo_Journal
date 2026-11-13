package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"todo-journal/database"

	"github.com/google/uuid"
)

type Todo struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Title       string  `json:"title"`
	Completed   bool    `json:"completed"`
	Priority    string  `json:"priority"`
	ReminderAt  *string `json:"reminder_at"`
	CompletedAt *string `json:"completed_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type CreateTodoRequest struct {
	Title      string  `json:"title"`
	Priority   string  `json:"priority"`
	ReminderAt *string `json:"reminder_at,omitempty"`
}

type UpdateTodoRequest struct {
	Title       *string `json:"title,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	ReminderAt  *string `json:"reminder_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
}

// TodosHandler routes GET /api/todos and POST /api/todos
func TodosHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserContextKey).(*TokenClaims)
	if !ok || claims == nil {
		JSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getTodos(w, claims.UserID)
	case http.MethodPost:
		createTodo(w, r, claims.UserID)
	default:
		JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// TodoItemHandler routes PUT /api/todos/{id} and DELETE /api/todos/{id}
func TodoItemHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserContextKey).(*TokenClaims)
	if !ok || claims == nil {
		JSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract ID from path: /api/todos/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/todos/")
	todoID := strings.TrimSpace(path)
	if todoID == "" {
		JSONError(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		updateTodo(w, r, claims.UserID, todoID)
	case http.MethodDelete:
		deleteTodo(w, claims.UserID, todoID)
	default:
		JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getTodos(w http.ResponseWriter, userID string) {
	// Active items (completed = 0) appear first (newest to oldest),
	// followed by completed items (completed = 1) at the bottom.
	rows, err := database.DB.Query(`
		SELECT id, user_id, title, completed, priority, reminder_at, completed_at, created_at, updated_at
		FROM todos
		WHERE user_id = ?
		ORDER BY completed ASC, created_at DESC
	`, userID)
	if err != nil {
		JSONError(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	todos := make([]Todo, 0)
	for rows.Next() {
		var t Todo
		var compInt int
		var reminderAtNull, completedAtNull sql.NullString
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &compInt, &t.Priority, &reminderAtNull, &completedAtNull, &createdAtStr, &updatedAtStr); err != nil {
			JSONError(w, "Failed to parse task data", http.StatusInternalServerError)
			return
		}

		t.Completed = compInt == 1
		t.CreatedAt = createdAtStr
		t.UpdatedAt = updatedAtStr

		if reminderAtNull.Valid && reminderAtNull.String != "" {
			val := reminderAtNull.String
			t.ReminderAt = &val
		}
		if completedAtNull.Valid && completedAtNull.String != "" {
			val := completedAtNull.String
			t.CompletedAt = &val
		}

		todos = append(todos, t)
	}

	JSONResponse(w, http.StatusOK, todos)
}

func createTodo(w http.ResponseWriter, r *http.Request, userID string) {
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		JSONError(w, "Task title cannot be empty", http.StatusBadRequest)
		return
	}

	priority := strings.ToLower(strings.TrimSpace(req.Priority))
	if priority != "low" && priority != "medium" && priority != "high" {
		priority = "medium"
	}

	var reminderAt *string
	if req.ReminderAt != nil {
		trimmed := strings.TrimSpace(*req.ReminderAt)
		if trimmed != "" {
			reminderAt = &trimmed
		}
	}

	todoID := uuid.New().String()
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err := database.DB.Exec(`
		INSERT INTO todos (id, user_id, title, completed, priority, reminder_at, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?, NULL, ?, ?)
	`, todoID, userID, req.Title, priority, reminderAt, now, now)

	if err != nil {
		JSONError(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	createdTodo := Todo{
		ID:          todoID,
		UserID:      userID,
		Title:       req.Title,
		Completed:   false,
		Priority:    priority,
		ReminderAt:  reminderAt,
		CompletedAt: nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	JSONResponse(w, http.StatusCreated, createdTodo)
}

func updateTodo(w http.ResponseWriter, r *http.Request, userID, todoID string) {
	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify existence and ownership
	var currentTitle, currentPriority string
	var currentCompInt int
	var currentReminderAt, currentCompletedAt sql.NullString
	var currentCreatedAt string

	err := database.DB.QueryRow(`
		SELECT title, completed, priority, reminder_at, completed_at, created_at
		FROM todos
		WHERE id = ? AND user_id = ?
	`, todoID, userID).Scan(&currentTitle, &currentCompInt, &currentPriority, &currentReminderAt, &currentCompletedAt, &currentCreatedAt)

	if err == sql.ErrNoRows {
		JSONError(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		JSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	newTitle := currentTitle
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			JSONError(w, "Task title cannot be empty", http.StatusBadRequest)
			return
		}
		newTitle = trimmed
	}

	newCompleted := currentCompInt
	var newCompletedAt *string
	if currentCompletedAt.Valid && currentCompletedAt.String != "" {
		val := currentCompletedAt.String
		newCompletedAt = &val
	}

	if req.Completed != nil {
		if *req.Completed {
			newCompleted = 1
			nowStr := time.Now().Format("2006-01-02 15:04:05")
			newCompletedAt = &nowStr
		} else {
			newCompleted = 0
			newCompletedAt = nil
		}
	} else if req.CompletedAt != nil {
		trimmed := strings.TrimSpace(*req.CompletedAt)
		if trimmed == "" {
			newCompletedAt = nil
		} else {
			newCompletedAt = &trimmed
		}
	}

	newPriority := currentPriority
	if req.Priority != nil {
		p := strings.ToLower(strings.TrimSpace(*req.Priority))
		if p == "low" || p == "medium" || p == "high" {
			newPriority = p
		}
	}

	var newReminderAt *string
	if currentReminderAt.Valid && currentReminderAt.String != "" {
		val := currentReminderAt.String
		newReminderAt = &val
	}

	if req.ReminderAt != nil {
		trimmed := strings.TrimSpace(*req.ReminderAt)
		if trimmed == "" {
			newReminderAt = nil
		} else {
			newReminderAt = &trimmed
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = database.DB.Exec(`
		UPDATE todos
		SET title = ?, completed = ?, priority = ?, reminder_at = ?, completed_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, newTitle, newCompleted, newPriority, newReminderAt, newCompletedAt, now, todoID, userID)

	if err != nil {
		JSONError(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	updatedTodo := Todo{
		ID:          todoID,
		UserID:      userID,
		Title:       newTitle,
		Completed:   newCompleted == 1,
		Priority:    newPriority,
		ReminderAt:  newReminderAt,
		CompletedAt: newCompletedAt,
		CreatedAt:   currentCreatedAt,
		UpdatedAt:   now,
	}

	JSONResponse(w, http.StatusOK, updatedTodo)
}

func deleteTodo(w http.ResponseWriter, userID, todoID string) {
	result, err := database.DB.Exec(`
		DELETE FROM todos WHERE id = ? AND user_id = ?
	`, todoID, userID)

	if err != nil {
		JSONError(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		JSONError(w, "Task not found", http.StatusNotFound)
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Task successfully deleted",
		"id":      todoID,
	})
}
