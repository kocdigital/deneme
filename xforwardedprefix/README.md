# X-Forwarded-Prefix Example

This example demonstrates how the `X-Forwarded-Prefix` header works in a reverse proxy scenario.

## What is X-Forwarded-Prefix?

`X-Forwarded-Prefix` is a non-standard HTTP header used in proxy environments to inform the backend server about the URL path prefix that the client originally used to connect to the proxy.

### Use Cases

1. **Path-Based Routing**: When a proxy routes requests to different backends based on URL paths
2. **Service Composition**: When multiple services are exposed under a unified path structure
3. **Preserving Path Context**: When backend services need to know their context in the full URL structure

## How This Example Works

This example consists of two servers:

1. **Backend Server** (port 8080): A simple HTTP server that displays request details
2. **Proxy Server** (port 9090): A reverse proxy that routes requests to the backend based on path

The proxy handles paths in different ways:

- `/api/*` routes: Strips the `/api` prefix and adds `X-Forwarded-Prefix: /api` header
- `/app/*` routes: Strips the `/app` prefix and adds `X-Forwarded-Prefix: /app` header
- `/v1/*` routes: Strips the `/v1` prefix and adds `X-Forwarded-Prefix: /v1` header
- `/direct` route: Proxies to backend without modifying paths or adding the header

## Running the Example

```bash
go run main.go
```

Then access:

- http://localhost:9090/ (Home page with instructions)
- http://localhost:9090/api/test (Proxied with X-Forwarded-Prefix)
- http://localhost:9090/app/users (Proxied with X-Forwarded-Prefix)
- http://localhost:9090/v1/data (Proxied with X-Forwarded-Prefix)
- http://localhost:9090/direct (Proxied without X-Forwarded-Prefix)

## Key Concepts

### 1. Path Stripping

The proxy strips the prefix from the request URL path before forwarding:

```
Original URL: /api/test
Backend receives: /test (with X-Forwarded-Prefix: /api)
```

### 2. Context Preservation

The backend application can use the `X-Forwarded-Prefix` header to reconstruct the original path if needed:

```
Backend sees: /test
X-Forwarded-Prefix: /api
Full path: /api/test
```

### 3. Relative URLs

When your backend generates links, it should consider the `X-Forwarded-Prefix` to ensure links work properly when served through the proxy.

## Common Mistakes

1. Not stripping the prefix from the request path
2. Not adding the `X-Forwarded-Prefix` header
3. Ignoring the `X-Forwarded-Prefix` in the backend when generating URLs

## Security Considerations

Since this header is typically used in internal infrastructures, you should validate or sanitize this header if your application is directly exposed to the internet, as malicious clients could set this header to attempt path traversal or other attacks.
