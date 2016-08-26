## some tools written in golang

### http-auth

Help you establish a http proxy with Http Basic Auth to other http proxies which is not supported.

Connections which pass the authentication will just forwarded to next http proxy.

```
Usage of ./http-auth:
  -addr string
    	bind address (default "0.0.0.0")
  -p int
    	bind port (default 9999)
  -proxy_addr string
    	redirect proxy address (default "127.0.0.1:8080")
  -pwd string
    	auth passwd (default "admin")
  -user string
    	auth username (default "admin")
```
