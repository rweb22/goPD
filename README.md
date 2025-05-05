# goPD

cmd 1:
```
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=*.github.dev"
```

cmd 2:
```
docker build -t gopd .
```

cmd 3:
```
docker run -d -p 8080:8080 -v $(pwd)/../pddata:/data gopd
```
