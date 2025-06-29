#!/bin/bash

# Test script for new rate distribution features
# Make sure to have the master running on port 8080

API_BASE="http://localhost:8080/api"
AUTH_URL="http://localhost:8080/auth/login"

# Login and get token
echo "Getting authentication token..."
TOKEN_RESPONSE=$(curl -s -X POST "$AUTH_URL" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}')

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.token')
if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "Failed to get token: $TOKEN_RESPONSE"
    exit 1
fi

echo "Token obtained: ${TOKEN:0:20}..."

# Test targets
TARGETS='[{"method":"GET","url":"https://dummyjson.com/http/200/test"}]'
TARGETS_BASE64=$(echo "$TARGETS" | base64)

echo ""
echo "=== Testing Rate Distribution Modes ==="

# Test 1: Shared distribution (default)
echo ""
echo "1. Testing SHARED distribution (30 req/s across 3 workers = ~10 req/s each)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Shared Distribution",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "shared",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

# Test 2: Same distribution
echo ""
echo "2. Testing SAME distribution (30 req/s per worker × 3 workers = 90 req/s total)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Same Distribution",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "same",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

# Test 3: Weighted distribution
echo ""
echo "3. Testing WEIGHTED distribution (30 req/s with weights [2.0, 1.0, 1.0] = 15, 7.5, 7.5 req/s)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Weighted Distribution",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "weighted",
    "rate_weights": [2.0, 1.0, 1.0],
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

# Test 4: Ramped distribution
echo ""
echo "4. Testing RAMPED distribution (30 req/s gradually increased across 3 workers)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Ramped Distribution",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "ramped",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

# Test 5: Burst distribution
echo ""
echo "5. Testing BURST distribution (70% load on first workers, 30% on remainder)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Burst Distribution",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "burst",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

echo ""
echo "=== Testing Error Cases ==="

# Test 6: Invalid distribution mode
echo ""
echo "6. Testing invalid distribution mode (should fail)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Invalid Distribution",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "invalid",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

# Test 7: Weighted distribution without weights
echo ""
echo "7. Testing weighted distribution without weights (should fail)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Weighted Without Weights",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "weighted",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

# Test 8: Weighted distribution with wrong number of weights
echo ""
echo "8. Testing weighted distribution with wrong number of weights (should fail)"
curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Weighted Wrong Count",
    "duration_seconds": "10s",
    "rate_per_second": 30,
    "worker_count": 3,
    "rate_distribution": "weighted",
    "rate_weights": [1.0, 2.0],
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }' | jq '.'

echo ""
echo "All tests completed! Check the logs to see the rate distribution in action."
