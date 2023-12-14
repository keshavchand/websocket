# Implement a websocket client in Golang

# Connection Upgrade Request

## Request
```
GET / HTTP/1.1
Host: localhost:8000
Connection: Upgrade
Pragma: no-cache
Cache-Control: no-cache
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36
Upgrade: websocket
Origin: chrome://newtab
Sec-WebSocket-Version: 13
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.9
Sec-WebSocket-Key: 4tLev7yozRKU6NvzdUj+yQ==
Sec-WebSocket-Extensions: permessage-deflate; client_max_window_bits
```

## Response
```
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: upgrade
Sec-WebSocket-Accept: OWQ4NjdjMDFhN2FiOGUwYzA4ZjYxNDc5MWFlMDQ2ZTViZDA1OWIzOA==
```
