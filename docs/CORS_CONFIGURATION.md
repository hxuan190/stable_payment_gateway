# CORS Configuration Guide

## Overview

Cross-Origin Resource Sharing (CORS) is a security mechanism that allows or restricts web applications running on one origin to access resources from a different origin. This is critical for the Payment Gateway as the frontend (merchant dashboard, admin panel) will be hosted on different domains from the API server.

## Why CORS Matters

Without proper CORS configuration:
- Merchant dashboards cannot make API calls to the payment gateway
- Admin panels cannot access admin endpoints
- Browser security will block all cross-origin requests
- Development and testing become difficult

## CORS Implementation

The Payment Gateway uses the `gin-contrib/cors` middleware to handle CORS. The configuration is applied globally to all routes in the API server.

### Configuration Location

CORS is configured in `/internal/api/server.go` in the `corsMiddleware()` function (lines 82-114).

## Environment Configuration

### Environment Variable

Configure allowed origins using the `API_ALLOW_ORIGINS` environment variable:

```bash
# Single origin
API_ALLOW_ORIGINS=http://localhost:3000

# Multiple origins (comma-separated)
API_ALLOW_ORIGINS=http://localhost:3000,http://localhost:3001,https://dashboard.payment-gateway.vn
```

### Configuration File

In `.env` or `.env.example`:

```bash
# Development - Allow local development servers
API_ALLOW_ORIGINS=http://localhost:3000,http://localhost:3001

# Staging
API_ALLOW_ORIGINS=https://dashboard-staging.payment-gateway.vn,https://admin-staging.payment-gateway.vn

# Production - Only allow production domains
API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn,https://admin.payment-gateway.vn
```

## CORS Settings

### Allowed Origins
- **Development**: `http://localhost:3000`, `http://localhost:3001` (local development)
- **Production**: Your production domain(s) only
- **Default**: `http://localhost:3000` if not specified

### Allowed Methods
The following HTTP methods are allowed:
- `GET` - Read operations
- `POST` - Create operations
- `PUT` - Full update operations
- `PATCH` - Partial update operations
- `DELETE` - Delete operations
- `OPTIONS` - Preflight requests

### Allowed Headers
The following request headers are allowed:
- `Origin` - Origin of the request
- `Content-Type` - Request content type (e.g., `application/json`)
- `Accept` - Acceptable response content types
- `Authorization` - API key or JWT token
- `X-Request-ID` - Request tracking ID

### Exposed Headers
The following response headers are exposed to the client:
- `Content-Length` - Response size
- `X-Request-ID` - Request tracking ID
- `X-RateLimit-Limit` - Rate limit maximum
- `X-RateLimit-Remaining` - Remaining requests
- `X-RateLimit-Reset` - Rate limit reset time
- `Retry-After` - When to retry (for rate limiting)

### Credentials
- **Allowed**: Yes (`Access-Control-Allow-Credentials: true`)
- This enables sending cookies, authorization headers, and TLS client certificates

### Max Age
- **Duration**: 12 hours (43200 seconds)
- Browsers cache preflight responses for this duration
- Reduces preflight request overhead

## Security Best Practices

### 1. Environment-Specific Origins

**Never use wildcard (`*`) in production!**

```bash
# ❌ BAD - Too permissive
API_ALLOW_ORIGINS=*

# ✅ GOOD - Explicit origins only
API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn
```

### 2. Use HTTPS in Production

```bash
# ❌ BAD - Insecure HTTP in production
API_ALLOW_ORIGINS=http://dashboard.payment-gateway.vn

# ✅ GOOD - Secure HTTPS only
API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn
```

### 3. Separate Development and Production Origins

```bash
# Development .env
API_ALLOW_ORIGINS=http://localhost:3000,http://localhost:3001

# Production .env
API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn,https://admin.payment-gateway.vn
```

### 4. Limit Origins to Necessary Domains

Only add origins that actually need access:

```bash
# ✅ GOOD - Only necessary domains
API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn,https://admin.payment-gateway.vn

# ❌ BAD - Too many unnecessary domains
API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn,https://admin.payment-gateway.vn,https://example.com,https://random-site.com
```

## Testing CORS

### Manual Testing with cURL

#### 1. Test Simple Request

```bash
curl -i -X GET http://localhost:8080/health \
  -H "Origin: http://localhost:3000"
```

Expected response headers:
```
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Credentials: true
```

#### 2. Test Preflight Request

```bash
curl -i -X OPTIONS http://localhost:8080/api/v1/payments \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type,Authorization"
```

Expected response:
```
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET,POST,PUT,PATCH,DELETE,OPTIONS
Access-Control-Allow-Headers: Origin,Content-Type,Accept,Authorization,X-Request-ID
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 43200
```

#### 3. Test Disallowed Origin

```bash
curl -i -X GET http://localhost:8080/health \
  -H "Origin: http://malicious-site.com"
```

Expected: No `Access-Control-Allow-Origin` header for disallowed origin.

### Automated Testing

Run the CORS test suite:

```bash
# Run all CORS tests
go test -v ./internal/api -run TestCORS

# Run specific test
go test -v ./internal/api -run TestCORS_AllowedOrigin

# Run with coverage
go test -v -cover ./internal/api -run TestCORS
```

### Browser Testing

#### JavaScript Fetch Example

```javascript
// Frontend code (running on http://localhost:3000)
fetch('http://localhost:8080/api/v1/payments', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'your-api-key'
  },
  credentials: 'include', // Include cookies
  body: JSON.stringify({
    amount_vnd: 100000,
    description: 'Test payment'
  })
})
  .then(response => response.json())
  .then(data => console.log(data))
  .catch(error => console.error('CORS error:', error));
```

## Common CORS Issues and Solutions

### Issue 1: "No 'Access-Control-Allow-Origin' header present"

**Cause**: Origin not in allowed list

**Solution**: Add origin to `API_ALLOW_ORIGINS`
```bash
API_ALLOW_ORIGINS=http://localhost:3000,http://localhost:3001
```

### Issue 2: "Preflight request doesn't pass access control check"

**Cause**: Requested headers or methods not allowed

**Solution**: Verify headers and methods are in allowed lists (check server.go configuration)

### Issue 3: "Credentials flag is true, but Access-Control-Allow-Credentials is false"

**Cause**: Credentials not enabled in CORS config

**Solution**: Already enabled in our config (`AllowCredentials: true`). Verify configuration is loaded correctly.

### Issue 4: "Origin not allowed in production"

**Cause**: Production environment not properly configured

**Solution**:
```bash
# Check current origins
echo $API_ALLOW_ORIGINS

# Set production origins
export API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn,https://admin.payment-gateway.vn
```

### Issue 5: "CORS works in development but not production"

**Cause**: Different origins in development vs production

**Solution**: Update production `.env` with production URLs:
```bash
# Production .env
ENV=production
API_ALLOW_ORIGINS=https://dashboard.payment-gateway.vn,https://admin.payment-gateway.vn
```

## Frontend Configuration Examples

### React/Next.js

```typescript
// lib/api.ts
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export async function createPayment(data: PaymentRequest) {
  const response = await fetch(`${API_BASE_URL}/api/v1/payments`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getApiKey()}`,
    },
    credentials: 'include', // Important for CORS with credentials
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw new Error('Failed to create payment');
  }

  return response.json();
}
```

### Vue.js

```javascript
// services/api.js
import axios from 'axios';

const api = axios.create({
  baseURL: process.env.VUE_APP_API_URL || 'http://localhost:8080',
  withCredentials: true, // Enable CORS credentials
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add API key to all requests
api.interceptors.request.use((config) => {
  const apiKey = localStorage.getItem('api_key');
  if (apiKey) {
    config.headers.Authorization = `Bearer ${apiKey}`;
  }
  return config;
});

export default api;
```

## Monitoring and Debugging

### Enable CORS Logging

To debug CORS issues, check the server logs for:
- Request origin
- CORS headers sent
- Configuration loaded

### Server Logs Example

```
INFO: HTTP request received
  origin: http://localhost:3000
  method: OPTIONS
  path: /api/v1/payments

INFO: CORS headers added
  Access-Control-Allow-Origin: http://localhost:3000
  Access-Control-Allow-Credentials: true
```

### Browser DevTools

1. Open browser DevTools (F12)
2. Go to Network tab
3. Make a request to the API
4. Check the request headers:
   - `Origin` should be present
5. Check the response headers:
   - `Access-Control-Allow-Origin` should match origin
   - `Access-Control-Allow-Credentials` should be `true`

### Preflight Request Debugging

In Network tab, look for OPTIONS requests before POST/PUT/DELETE:
- Status should be `204 No Content`
- Response time should be fast (< 100ms)
- CORS headers should be present

## Performance Considerations

### Preflight Caching

The `Access-Control-Max-Age: 43200` header tells browsers to cache preflight results for 12 hours, reducing:
- Preflight request overhead
- API server load
- Network latency

### Impact on API Performance

CORS middleware adds minimal overhead:
- Simple requests: ~0.1ms
- Preflight requests: ~0.5ms

Based on benchmarks (see `server_test.go`):
```
BenchmarkCORS_Middleware-8    500000    ~2500 ns/op
```

## Production Checklist

Before deploying to production, verify:

- [ ] `API_ALLOW_ORIGINS` contains only production domains
- [ ] All origins use HTTPS (not HTTP)
- [ ] No wildcard (`*`) in allowed origins
- [ ] Test CORS with actual production URLs
- [ ] Verify preflight requests work correctly
- [ ] Check browser console for CORS errors
- [ ] Monitor CORS-related errors in logs
- [ ] Document allowed origins for team
- [ ] Set up alerts for CORS failures

## References

- [MDN: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [gin-contrib/cors Documentation](https://github.com/gin-contrib/cors)
- [W3C CORS Specification](https://www.w3.org/TR/cors/)

## Support

If you encounter CORS issues not covered here:

1. Check server logs for CORS-related errors
2. Verify environment configuration
3. Test with cURL to isolate frontend vs backend issues
4. Review browser DevTools Network tab
5. Consult the team lead or senior developer

---

**Last Updated**: 2025-11-17
**Maintained By**: Development Team
