# goPD

cmd 1:
```
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=*.github.com"
```

cmd 2:
```
docker build -t goPD .
```

cmd 3:
```
docker run -d -p 8443:8443 -p 8080:8080 goPD
```
