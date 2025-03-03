# Stage 1: Build Go binary
FROM golang:1.21 AS builder
WORKDIR /app

# Copy all Go files
COPY *.go .

# Initialize Go module (replace <project_name> with your desired module name)
RUN go mod init goPD

# Download dependencies and tidy up
RUN go mod tidy

# Compile into a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 2: Set up Nginx and the app
FROM nginx:alpine
WORKDIR /app

# Copy the compiled Go binary
COPY --from=builder /app/server /app/server

# Copy static files
COPY dashboard /usr/share/nginx/html

# Copy Nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Copy SSL certificates
COPY cert.pem /etc/nginx/ssl/cert.pem
COPY key.pem /etc/nginx/ssl/key.pem

# Ensure SSL directory exists
RUN mkdir -p /etc/nginx/ssl

# Expose ports
EXPOSE 8443  # HTTPS port
EXPOSE 8080  # HTTP redirect port

# Run the Go server and Nginx
CMD ["/bin/sh", "-c", "./server & nginx -g 'daemon off;'"]