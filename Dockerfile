# Use an official Go runtime as a parent image
FROM golang:1.23.3 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# RUN git clone https://github.com/coinbase/x402.git /tmp/x402

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
# Using CGO_ENABLED=0 produces a statically linked binary that doesn't require libc
# This is good for small Alpine-based images
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/linkshrink ./cmd/server/main.go

# --- Final Stage ---

# Use a minimal base image for the final container
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/linkshrink .

# Copy potentially needed non-code files (e.g., .env, templates - adjust as needed)
COPY .env .
COPY templates/ /app/templates/

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/app/linkshrink"] 