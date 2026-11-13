# -------- Build Stage --------
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Install git (required for go modules that need it) and ca-certificates
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary (disable CGO for static binary)
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o todo-server ./

# -------- Runtime Stage --------
FROM alpine:3.20
WORKDIR /app

# Create a dedicated directory for the SQLite database
RUN mkdir -p /app/data

# Copy the satic files from the builder stage
COPY --from=builder /app/static ./static

# Copy the compiled binary from the builder stage
COPY --from=builder /app/todo-server .

# Expose the port the app listens on (assuming 8080, adjust if needed)
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["./todo-server"]
