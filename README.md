# ⚡ Personal Task Manager

A responsive, full-stack web-based task management application featuring **user authentication**, **SQLite database persistence**, automatic **bottom-sorting of completed tasks**, **inline text editing**, **delete confirmation prompts**, dynamic **Light/Dark logos**, and **Light / Dark mode switching**.

| Light Mode Logo | Dark Mode Logo |
| :---: | :---: |
| ![Light Logo](static/logo-light.jpg) | ![Dark Logo](static/logo-dark.jpg) |

---

## 🌟 Key Features

1. **User Authentication (Sign Up & Sign In)**:
   - Full account registration with email validation and encrypted passwords (bcrypt hashing).
   - Secure HMAC-SHA256 JWT tokens for stateless authentication.
   - User data isolation: each user's tasks are private to their account.
   - ⚡ **1-Click Demo Login** (`demo@example.com` / `demo123`) for rapid testing.

2. **Smart Task Management**:
   - **Reminders & Due Dates**: Set custom reminder date and time (`datetime-local`) with visual indicators for upcoming and ⚠️ **Overdue** tasks.
   - **Completion Timestamp**: Marking a task complete records its exact completion date and time (`completed_at`).
   - **Quick Add**: Input tasks with priority tags (Low, Medium, High) and optional reminder date/time using the Add button or pressing `Enter`.
   - **Automatic Bottom-Sorting**: Marking a task complete automatically reorders and sinks it to the bottom with visual strike-through styling.
   - **Inline Text Editing**: Double-click any task title or click the pencil icon to edit task title, priority, and reminder date/time in-place. Save with `Enter` or cancel with `Escape`.
   - **Delete Confirmation Modal**: Trash icon triggers an accessible confirmation prompt to prevent accidental deletions.
   - **Multi-Dimensional Filters**:
     - **Status Filter**: All, Active, Completed.
     - **Priority Filter**: All Priorities, 🔴 High, 🟡 Medium, 🟢 Low.
     - **Date Filter**: All Dates, 📅 Today, ⏰ Upcoming, ⚠️ Overdue, or custom date picker.
     - **Search Bar**: Real-time title query matching.
   - **Visual Progress Bar**: Dynamic completion counter (e.g. "4 of 6 tasks completed (67%)").

3. **Responsive Light & Dark Mode**:
   - Dedicated theme switch (Sun / Moon) persisted in `localStorage`.
   - Carefully tuned high-contrast slate and brand palette in Tailwind CSS.
   - Works across smartphones, tablets, and desktops.

4. **Public Internet & SQLite Persistence**:
   - Backed by **SQLite** (`todos.db`) with WAL mode enabled for high concurrency.
   - Self-contained single executable (`todo-server.exe`) that serves both the REST API and the React frontend.
   - Zero runtime dependencies to deploy to any cloud server or VPS.

---

## 🚀 How to Run Locally

### Option 1: Running the compiled binary directly
Double-click `todo-server.exe` in Windows Explorer, or run in terminal:
```powershell
.\todo-server.exe
```
Then open your web browser at:
👉 **`http://localhost:8080`**

### Option 2: Running with Go
```powershell
go run main.go
```

### Custom Port / Database Path:
You can configure the server with environment variables:
```powershell
$env:PORT="3000"
$env:DB_PATH="todos.db"
.\todo-server.exe
```

---

## 🌐 Hosting on the Public Internet

Because the application compiles into a single portable binary and uses SQLite:

1. **Deploying to a Linux VPS (Ubuntu/Debian)**:
   - Cross-compile for Linux from Windows:
     ```powershell
     $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o todo-server-linux .
     ```
   - Copy `todo-server-linux` and the `static/` directory to your VPS.
   - Run `chmod +x todo-server-linux && ./todo-server-linux` or manage with `systemd` / Docker.
2. **Reverse Proxy (Nginx / Caddy / Cloudflare)**:
   - Route domain `todo.yourdomain.com` to `localhost:8080` with HTTPS enabled.

---

## 🗄️ Database Schema

### `users`
- `id` (TEXT PRIMARY KEY - UUID)
- `name` (TEXT NOT NULL)
- `email` (TEXT UNIQUE NOT NULL COLLATE NOCASE)
- `password_hash` (TEXT NOT NULL - bcrypt)
- `created_at` (DATETIME DEFAULT CURRENT_TIMESTAMP)

### `todos`
- `id` (TEXT PRIMARY KEY - UUID)
- `user_id` (TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE)
- `title` (TEXT NOT NULL)
- `completed` (INTEGER DEFAULT 0)
- `priority` (TEXT DEFAULT 'medium' - 'low' | 'medium' | 'high')
- `reminder_at` (TEXT / DATETIME - Optional due/reminder timestamp)
- `completed_at` (TEXT / DATETIME - Timestamp recorded upon completion)
- `created_at` (DATETIME DEFAULT CURRENT_TIMESTAMP)
- `updated_at` (DATETIME DEFAULT CURRENT_TIMESTAMP)

---

## 💻 Tech Stack
- **Frontend**: React 18, Tailwind CSS, Lucide SVG Icons.
- **Backend**: Go 1.26, SQLite (`modernc.org/sqlite`), bcrypt (`golang.org/x/crypto`), HMAC-SHA256 JWT.
