# Build stage
FROM golang:1.25.7-alpine AS builder

WORKDIR /app

# Install dependencies required for some CGO/build operations or debugging if needed
# We also install timezone data
RUN apk add --no-cache git tzdata

# Cache Go modules first
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build the application statically
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o weiclothe-api ./cmd/api/main.go

# Final stage - using a minimal alpine image
FROM alpine:latest

WORKDIR /app

# Install CA certificates (needed for Keycloak/HTTPS external calls) and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/weiclothe-api .

# Expose the port (Mainly documentation, but matches HTTP_ADDR from your main.go)
EXPOSE 8080

# Run the executable
CMD ["./weiclothe-api"]
