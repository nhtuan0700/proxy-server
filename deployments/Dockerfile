# --- Build Stage ---
FROM golang:1.24.5 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app

# --- Run Stage ---
FROM alpine:latest

# Install SSL certificates (optional, if your app uses HTTPS)
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the compiled binary from builder stage
COPY --from=builder /app/app .

# Command to run
ENTRYPOINT ["./app"]
