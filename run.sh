docker build -t gopd .
docker run -d -p 8080:8080 -v $(pwd)/dashboard:/app/dashboard -v $(pwd)/../pddata:/data gopd
