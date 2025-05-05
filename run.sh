docker build -t gopd .
docker run -d -p 8080:8080 -v $(pwd)/../pddata:/data gopd
