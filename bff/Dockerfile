# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o bff cmd/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS requests if needed
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/bff .

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["./bff"]
