# Stage 1: Build Go binary
FROM golang:1.21 AS builder
WORKDIR /app

# Copy Go files, a.txt, and initialize module
COPY *.go ./
COPY holidays.txt ./
RUN go mod init goPD
RUN go mod tidy

# Compile into a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 2: Final image with Go server, time zone data, and a.txt
FROM alpine:latest
WORKDIR /app

# Install tzdata for time zone support
RUN apk add --no-cache tzdata

# Copy the compiled Go binary
COPY --from=builder /app/server /app/server

# Copy static files to be served by the Go server
COPY dashboard /app/dashboard

# Copy a.txt for the Go server to read
COPY holidays.txt /app/holidays.txt

VOLUME ["/data"]

# Expose the port your Go server listens on (e.g., 8080)
EXPOSE 8080

# Run the Go server
CMD ["./server"]
