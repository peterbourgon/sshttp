# sshttp

Problem: you're connected to a network that restricts outgoing connections to
port 80. This makes it impossible to SSH to your boxes.

Solution: run `sshttp` on your box bound to port 80. It peeks at the first few
bytes of each incoming connection to determine if it's SSH or HTTP, and proxies
the connection to the right place.

```
server$ sudo sshttp -listen :80 -http localhost:8080 -ssh localhost:22
2017/12/12 15:01:02 listening on :80
2017/12/12 15:01:02 proxying HTTP to localhost:8080
2017/12/12 15:01:02 proxying SSH to localhost:22
```

```
laptop$ curl -XGET http://server
<html>
<head><title>Welcome to my website</title></head>
<body>
<h1>My website!</h1>
</body>
</html>
```

```
laptop$ ssh -p 80 server
server$ _
```

