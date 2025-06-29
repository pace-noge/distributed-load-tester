# 🔌 API Reference - Test Submission

This guide shows how to submit tests programmatically using the REST API.

## 🚀 Authentication

First, get an authentication token:

```bash
curl -X POST "http://localhost:8080/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {"username": "admin"}
}
```

## 📝 Basic Test Submission

### Simple GET Request Test

```bash
curl -X POST "http://localhost:8080/api/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "name": "API Health Check",
    "duration_seconds": "30s",
    "rate_per_second": 50,
    "worker_count": 3,
    "rate_distribution": "shared",
    "targets_base64": "W3sibWV0aG9kIjoiR0VUIiwidXJsIjoiaHR0cHM6Ly9hcGkuZXhhbXBsZS5jb20vaGVhbHRoIn1d",
    "vegeta_payload_json": "{}"
  }'
```

### POST Request with Body

```bash
# First, prepare your targets (array of HTTP requests)
TARGETS='[{
  "method": "POST",
  "url": "https://api.example.com/login",
  "header": {
    "Content-Type": ["application/json"]
  },
  "body": "{\"username\":\"test\",\"password\":\"test123\"}"
}]'

# Encode targets to base64
TARGETS_BASE64=$(echo "$TARGETS" | base64)

# Submit the test
curl -X POST "http://localhost:8080/api/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "name": "Login API Test",
    "duration_seconds": "2m",
    "rate_per_second": 25,
    "worker_count": 3,
    "rate_distribution": "shared",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }'
```

## 📋 Request Parameters

### Required Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `name` | string | Test name | `"API Load Test"` |
| `duration_seconds` | string | Test duration | `"30s"`, `"2m"`, `"1h"` |
| `rate_per_second` | number | Requests per second | `50` |
| `worker_count` | number | Number of workers | `3` |
| `rate_distribution` | string | Distribution mode | `"shared"` |
| `targets_base64` | string | Base64 encoded targets | See below |
| `vegeta_payload_json` | string | Vegeta config (JSON) | `"{}"` |

### Optional Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `rate_weights` | array | Weights for weighted distribution | `[2.0, 1.0, 1.0]` |

### Rate Distribution Options

| Value | Description |
|-------|-------------|
| `"shared"` | Split rate evenly across workers |
| `"same"` | Each worker sends full rate |
| `"weighted"` | Use custom weights per worker |
| `"ramped"` | Gradually increasing per worker |
| `"burst"` | Front-loaded distribution |

## 🎯 Target Configuration

Targets define the HTTP requests to execute. They must be base64 encoded.

### Single GET Request
```json
[{
  "method": "GET",
  "url": "https://api.example.com/users"
}]
```

### POST with Headers and Body
```json
[{
  "method": "POST",
  "url": "https://api.example.com/users",
  "header": {
    "Content-Type": ["application/json"],
    "Authorization": ["Bearer token123"]
  },
  "body": "{\"name\":\"John\",\"email\":\"john@example.com\"}"
}]
```

### Multiple Endpoints (Random Selection)
```json
[
  {
    "method": "GET",
    "url": "https://api.example.com/users"
  },
  {
    "method": "GET",
    "url": "https://api.example.com/posts"
  },
  {
    "method": "GET",
    "url": "https://api.example.com/comments"
  }
]
```

## 🔧 Complete Examples

### Example 1: E-commerce API Test

```bash
#!/bin/bash

# Login and get token
TOKEN_RESPONSE=$(curl -s -X POST "http://localhost:8080/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}')

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.token')

# Define targets
TARGETS='[
  {"method": "GET", "url": "https://shop.example.com/api/products"},
  {"method": "GET", "url": "https://shop.example.com/api/categories"},
  {
    "method": "POST",
    "url": "https://shop.example.com/api/cart/add",
    "header": {"Content-Type": ["application/json"]},
    "body": "{\"product_id\":123,\"quantity\":1}"
  }
]'

TARGETS_BASE64=$(echo "$TARGETS" | base64)

# Submit test
curl -X POST "http://localhost:8080/api/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "E-commerce API Load Test",
    "duration_seconds": "5m",
    "rate_per_second": 100,
    "worker_count": 3,
    "rate_distribution": "shared",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }'
```

### Example 2: Authentication Stress Test

```bash
#!/bin/bash

API_BASE="http://localhost:8080"

# Get token
TOKEN=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' | jq -r '.token')

# Test login endpoint with high load
TARGETS='[{
  "method": "POST",
  "url": "https://api.example.com/auth/login",
  "header": {"Content-Type": ["application/json"]},
  "body": "{\"username\":\"loadtest\",\"password\":\"password123\"}"
}]'

TARGETS_BASE64=$(echo "$TARGETS" | base64)

# Use "same" distribution for stress testing
curl -X POST "$API_BASE/api/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Auth Stress Test",
    "duration_seconds": "3m",
    "rate_per_second": 50,
    "worker_count": 3,
    "rate_distribution": "same",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }'
```

### Example 3: Weighted Distribution Test

```bash
#!/bin/bash

API_BASE="http://localhost:8080"

# Get token
TOKEN=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' | jq -r '.token')

# Test with weighted distribution (simulating geographic load)
TARGETS='[{"method": "GET", "url": "https://api.example.com/data"}]'
TARGETS_BASE64=$(echo "$TARGETS" | base64)

curl -X POST "$API_BASE/api/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Geographic Load Simulation",
    "duration_seconds": "4m",
    "rate_per_second": 120,
    "worker_count": 3,
    "rate_distribution": "weighted",
    "rate_weights": [2.0, 1.5, 1.0],
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }'
```

## 📊 Response Format

### Success Response
```json
{
  "message": "Test submitted successfully",
  "testId": "af99ea66-ac35-4843-8537-2e72c1149c77"
}
```

### Error Response
```json
{
  "error": "Invalid request format",
  "details": "rate_per_second must be greater than 0"
}
```

## 🔍 Checking Test Status

### Get Test Status
```bash
curl -X GET "http://localhost:8080/api/test/af99ea66-ac35-4843-8537-2e72c1149c77" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Response
```json
{
  "test_id": "af99ea66-ac35-4843-8537-2e72c1149c77",
  "name": "API Load Test",
  "status": "COMPLETED",
  "assigned_workers": 3,
  "completed_workers": 3,
  "failed_workers": 0,
  "created_at": "2025-06-30T03:15:30Z",
  "updated_at": "2025-06-30T03:17:45Z"
}
```

## 📈 Getting Test Results

### Get Aggregated Results
```bash
curl -X GET "http://localhost:8080/api/test/af99ea66-ac35-4843-8537-2e72c1149c77/results" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Response
```json
{
  "test_id": "af99ea66-ac35-4843-8537-2e72c1149c77",
  "total_requests": 15000,
  "successful_requests": 14850,
  "failed_requests": 150,
  "success_rate": 0.99,
  "average_latency_ms": 125.5,
  "p95_latency_ms": 245.0,
  "p99_latency_ms": 389.2,
  "total_duration_ms": 180000,
  "requests_per_second": 83.33,
  "status_code_distribution": {
    "200": 14850,
    "429": 100,
    "500": 50
  },
  "created_at": "2025-06-30T03:17:45Z"
}
```

## 🛠️ Helper Scripts

### Base64 Encoding Helper

```bash
#!/bin/bash
# encode_targets.sh

if [ $# -eq 0 ]; then
    echo "Usage: $0 'JSON_TARGETS'"
    echo "Example: $0 '[{\"method\":\"GET\",\"url\":\"https://api.example.com\"}]'"
    exit 1
fi

echo "$1" | base64
```

### Complete Test Runner

```bash
#!/bin/bash
# run_load_test.sh

set -e

API_BASE="${API_BASE:-http://localhost:8080}"
USERNAME="${USERNAME:-admin}"
PASSWORD="${PASSWORD:-password}"

# Function to get auth token
get_token() {
    curl -s -X POST "$API_BASE/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" | \
        jq -r '.token'
}

# Function to submit test
submit_test() {
    local name="$1"
    local url="$2"
    local method="${3:-GET}"
    local rate="${4:-50}"
    local duration="${5:-30s}"
    local workers="${6:-3}"

    local token=$(get_token)
    local targets="[{\"method\":\"$method\",\"url\":\"$url\"}]"
    local targets_base64=$(echo "$targets" | base64)

    curl -s -X POST "$API_BASE/api/test/submit" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{
            \"name\": \"$name\",
            \"duration_seconds\": \"$duration\",
            \"rate_per_second\": $rate,
            \"worker_count\": $workers,
            \"rate_distribution\": \"shared\",
            \"targets_base64\": \"$targets_base64\",
            \"vegeta_payload_json\": \"{}\"
        }" | jq .
}

# Example usage
submit_test "Quick API Test" "https://httpbin.org/get" "GET" 25 "1m" 3
```

## 🔒 Security Notes

1. **Protect your auth tokens**: Never commit tokens to version control
2. **Use environment variables**: Store credentials securely
3. **Token expiration**: Tokens expire, implement refresh logic
4. **Rate limiting**: Be respectful of target services
5. **HTTPS only**: Always use HTTPS in production

## 🚨 Error Codes

| HTTP Code | Meaning | Solution |
|-----------|---------|----------|
| 400 | Bad Request | Check request format |
| 401 | Unauthorized | Get new auth token |
| 403 | Forbidden | Check permissions |
| 422 | Validation Error | Fix request parameters |
| 500 | Server Error | Check server logs |

---

**Happy API Testing! 🚀**
