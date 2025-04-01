FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -o /app/bin/server ./cmd/server

# Create a minimal production image
FROM alpine:3.17

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/bin/server /app/server

# Expose the port
EXPOSE 8080

# Run the server
CMD ["/app/server"]