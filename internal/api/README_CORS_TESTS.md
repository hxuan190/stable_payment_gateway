# CORS Tests - Execution Guide

## Test File

The CORS tests are located in `/internal/api/server_test.go`.

## Running Tests

### Run All CORS Tests

```bash
go test -v ./internal/api -run TestCORS
```

### Run Specific Test

```bash
# Test allowed origins
go test -v ./internal/api -run TestCORS_AllowedOrigin

# Test preflight requests
go test -v ./internal/api -run TestCORS_PreflightRequest

# Test multiple origins
go test -v ./internal/api -run TestCORS_MultipleOrigins
```

### Run with Coverage

```bash
go test -v -cover ./internal/api -run TestCORS
go test -coverprofile=coverage.out ./internal/api -run TestCORS
go tool cover -html=coverage.out
```

### Run Benchmark

```bash
go test -bench=BenchmarkCORS ./internal/api
```

## Test Coverage

The CORS test suite includes:

1. **TestCORS_AllowedOrigin** - Verifies allowed origins receive CORS headers
2. **TestCORS_DisallowedOrigin** - Verifies disallowed origins are blocked
3. **TestCORS_PreflightRequest** - Tests OPTIONS preflight requests
4. **TestCORS_AllowedMethods** - Verifies all HTTP methods are allowed
5. **TestCORS_AllowedHeaders** - Tests request header allowlist
6. **TestCORS_ExposedHeaders** - Verifies response headers are exposed
7. **TestCORS_Credentials** - Tests credentials support
8. **TestCORS_MaxAge** - Verifies preflight cache duration
9. **TestCORS_MultipleOrigins** - Tests multiple allowed origins
10. **TestCORS_ProductionConfig** - Tests production-specific configuration
11. **TestCORS_ComplexRequest** - Tests complex CORS scenarios
12. **TestCORS_HealthEndpoint** - Verifies CORS on health check
13. **TestCORS_ConfigurationFromEnv** - Tests environment variable parsing
14. **TestCORS_CacheControlHeaders** - Tests header coexistence
15. **TestCORS_VaryHeader** - Tests Vary header for caching
16. **BenchmarkCORS_Middleware** - Performance benchmark

## Prerequisites

Before running tests, ensure:

1. Go 1.21+ is installed
2. All dependencies are downloaded: `go mod download`
3. Redis is not required for CORS tests (mocked if needed)
4. Database is not required for CORS tests (mocked if needed)

## Expected Results

All tests should pass with output similar to:

```
=== RUN   TestCORS_AllowedOrigin
--- PASS: TestCORS_AllowedOrigin (0.00s)
=== RUN   TestCORS_DisallowedOrigin
--- PASS: TestCORS_DisallowedOrigin (0.00s)
=== RUN   TestCORS_PreflightRequest
--- PASS: TestCORS_PreflightRequest (0.00s)
...
PASS
ok      github.com/hxuan190/stable_payment_gateway/internal/api 0.xyz s
```

## Troubleshooting

### Test Fails: "no such host"

**Cause**: Network issues during dependency download

**Solution**:
```bash
# Set GOPROXY to a working proxy
export GOPROXY=https://proxy.golang.org,direct
go mod download
```

### Test Fails: "package not found"

**Cause**: Missing dependencies

**Solution**:
```bash
go mod tidy
go mod download
```

### Test Fails: "undefined: ServerConfig"

**Cause**: Trying to run test file in isolation

**Solution**: Run tests from the correct directory
```bash
# From project root
go test ./internal/api -run TestCORS

# NOT this (will fail)
go test ./internal/api/server_test.go
```

## Continuous Integration

Add to your CI/CD pipeline:

```yaml
# Example GitHub Actions
- name: Run CORS Tests
  run: |
    go test -v -race ./internal/api -run TestCORS
    go test -cover ./internal/api -run TestCORS
```

## Next Steps

After tests pass:

1. ✅ Review test coverage report
2. ✅ Verify all edge cases are covered
3. ✅ Update documentation if needed
4. ✅ Run integration tests with real frontend
5. ✅ Deploy to staging and verify with actual domains

## Related Documentation

- [CORS Configuration Guide](/docs/CORS_CONFIGURATION.md)
- [API Documentation](/internal/api/README.md)
- [Server Implementation](/internal/api/server.go)

---

**Last Updated**: 2025-11-17
**Test Count**: 15 tests + 1 benchmark
