# Stage 1: Build Go binary
FROM golang:1.21 AS builder
WORKDIR /app

# Copy Go files and initialize module
COPY *.go .
RUN go mod init goPD
RUN go mod tidy

# Compile into a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 2: Final image with just the Go server
FROM alpine:latest
WORKDIR /app

# Copy the compiled Go binary
COPY --from=builder /app/server /app/server

# Copy static files to be served by the Go server
COPY dashboard /app/dashboard

# Expose the port your Go server listens on (e.g., 8081)
EXPOSE 8080

# Run the Go server
CMD ["./server"]
