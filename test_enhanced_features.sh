#!/bin/bash

# Comprehensive test script for the enhanced distributed load tester
# Tests pagination, replay functionality, detailed results, and UI features

API_BASE="http://localhost:8080/api"
AUTH_URL="http://localhost:8080/auth/login"

echo "=== Enhanced Distributed Load Tester Test Suite ==="
echo "Testing pagination, replay, detailed results, and enhanced features"
echo ""

# Login and get token
echo "1. Getting authentication token..."
TOKEN_RESPONSE=$(curl -s -X POST "$AUTH_URL" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}')

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.token')
if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token: $TOKEN_RESPONSE"
    exit 1
fi

echo "✅ Token obtained: ${TOKEN:0:20}..."
echo ""

# Test targets
TARGETS='[{"method":"GET","url":"https://dummyjson.com/http/200/test"}]'
TARGETS_BASE64=$(echo "$TARGETS" | base64)

echo "=== Testing Core Functionality ==="
echo ""

# Test 1: Create multiple tests for pagination testing
echo "2. Creating multiple tests for pagination testing..."
for i in {1..5}; do
    echo "Creating test $i..."
    curl -s -X POST "$API_BASE/test/submit" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d '{
        "name": "Pagination Test '"$i"'",
        "duration_seconds": "5s",
        "rate_per_second": 5,
        "worker_count": 1,
        "rate_distribution": "shared",
        "targets_base64": "'"$TARGETS_BASE64"'",
        "vegeta_payload_json": "{}"
      }' > /dev/null

    if [ $? -eq 0 ]; then
        echo "✅ Test $i created successfully"
    else
        echo "❌ Failed to create test $i"
    fi
    sleep 1
done
echo ""

# Test 2: Test pagination functionality
echo "3. Testing pagination functionality..."
echo "Getting page 1 with limit 3:"
PAGE1_RESPONSE=$(curl -s -X GET "$API_BASE/tests/history?page=1&limit=3" \
  -H "Authorization: Bearer $TOKEN")

echo "$PAGE1_RESPONSE" | jq -r '.tests[] | "  - " + .name + " (" + .status + ")"'
TOTAL_PAGES=$(echo "$PAGE1_RESPONSE" | jq -r '.total_pages')
echo "Total pages: $TOTAL_PAGES"
echo ""

if [ "$TOTAL_PAGES" -gt 1 ]; then
    echo "Getting page 2:"
    PAGE2_RESPONSE=$(curl -s -X GET "$API_BASE/tests/history?page=2&limit=3" \
      -H "Authorization: Bearer $TOKEN")
    echo "$PAGE2_RESPONSE" | jq -r '.tests[] | "  - " + .name + " (" + .status + ")"'
else
    echo "Only one page available"
fi
echo ""

# Test 3: Get test detail for the first test
echo "4. Testing test detail functionality..."
FIRST_TEST_ID=$(echo "$PAGE1_RESPONSE" | jq -r '.tests[0].id')
if [ "$FIRST_TEST_ID" != "null" ] && [ -n "$FIRST_TEST_ID" ]; then
    echo "Getting details for test: $FIRST_TEST_ID"
    DETAIL_RESPONSE=$(curl -s -X GET "$API_BASE/tests/$FIRST_TEST_ID" \
      -H "Authorization: Bearer $TOKEN")

    if echo "$DETAIL_RESPONSE" | jq -e '.test' > /dev/null; then
        echo "✅ Test detail retrieved successfully"
        echo "  Test name: $(echo "$DETAIL_RESPONSE" | jq -r '.test.name')"
        echo "  Rate distribution: $(echo "$DETAIL_RESPONSE" | jq -r '.test.rate_distribution')"
        echo "  Worker count: $(echo "$DETAIL_RESPONSE" | jq -r '.test.worker_count')"

        RESULT_COUNT=$(echo "$DETAIL_RESPONSE" | jq -r '.results | length')
        echo "  Results count: $RESULT_COUNT"

        HAS_AGGREGATED=$(echo "$DETAIL_RESPONSE" | jq -r '.aggregated_result != null')
        echo "  Has aggregated result: $HAS_AGGREGATED"
    else
        echo "❌ Failed to get test detail"
        echo "$DETAIL_RESPONSE"
    fi
else
    echo "❌ No test ID found for detail testing"
fi
echo ""

# Test 4: Test replay functionality
echo "5. Testing test replay functionality..."
if [ "$FIRST_TEST_ID" != "null" ] && [ -n "$FIRST_TEST_ID" ]; then
    echo "Replaying test: $FIRST_TEST_ID"
    REPLAY_RESPONSE=$(curl -s -X POST "$API_BASE/tests/$FIRST_TEST_ID/replay" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d '{"name": "Enhanced Replay Test"}')

    if echo "$REPLAY_RESPONSE" | jq -e '.id' > /dev/null; then
        REPLAY_TEST_ID=$(echo "$REPLAY_RESPONSE" | jq -r '.id')
        echo "✅ Test replayed successfully"
        echo "  Original test ID: $FIRST_TEST_ID"
        echo "  Replayed test ID: $REPLAY_TEST_ID"
        echo "  Replayed test name: $(echo "$REPLAY_RESPONSE" | jq -r '.name')"
    else
        echo "❌ Failed to replay test"
        echo "$REPLAY_RESPONSE"
    fi
else
    echo "❌ No test ID available for replay testing"
fi
echo ""

# Test 5: Test enhanced rate distribution modes
echo "6. Testing enhanced rate distribution modes..."

# Weighted distribution test
echo "Creating weighted distribution test..."
WEIGHTED_RESPONSE=$(curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Enhanced Weighted Test",
    "duration_seconds": "8s",
    "rate_per_second": 20,
    "worker_count": 3,
    "rate_distribution": "weighted",
    "rate_weights": [2.0, 1.5, 0.5],
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }')

if echo "$WEIGHTED_RESPONSE" | jq -e '.id' > /dev/null; then
    echo "✅ Weighted distribution test created"
else
    echo "❌ Failed to create weighted test"
fi

# Enhanced ramped distribution test
echo "Creating enhanced ramped distribution test..."
RAMPED_RESPONSE=$(curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Enhanced Ramped Test",
    "duration_seconds": "15s",
    "rate_per_second": 30,
    "worker_count": 2,
    "rate_distribution": "ramped",
    "ramp_duration": "10s",
    "ramp_start_delay": "2s",
    "ramp_steps": 4,
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }')

if echo "$RAMPED_RESPONSE" | jq -e '.id' > /dev/null; then
    echo "✅ Enhanced ramped distribution test created"
else
    echo "❌ Failed to create enhanced ramped test"
fi
echo ""

# Test 6: Test enhanced Vegeta options
echo "7. Testing enhanced Vegeta options..."
VEGETA_RESPONSE=$(curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Enhanced Vegeta Options Test",
    "duration_seconds": "6s",
    "rate_per_second": 15,
    "worker_count": 1,
    "rate_distribution": "shared",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{\"timeout\": 20, \"redirects\": 8, \"keepalive\": true, \"http2\": false, \"insecure\": false, \"connections\": 100, \"headers\": {\"User-Agent\": \"EnhancedLoadTester/2.0\", \"X-Test-Mode\": \"enhanced\"}}"
  }')

if echo "$VEGETA_RESPONSE" | jq -e '.id' > /dev/null; then
    echo "✅ Enhanced Vegeta options test created"
    VEGETA_TEST_ID=$(echo "$VEGETA_RESPONSE" | jq -r '.id')
    echo "  Test ID: $VEGETA_TEST_ID"
else
    echo "❌ Failed to create enhanced Vegeta test"
fi
echo ""

# Test 7: Dashboard functionality
echo "8. Testing dashboard functionality..."
DASHBOARD_RESPONSE=$(curl -s -X GET "$API_BASE/dashboard" \
  -H "Authorization: Bearer $TOKEN")

if echo "$DASHBOARD_RESPONSE" | jq -e '.total_workers' > /dev/null; then
    echo "✅ Dashboard data retrieved successfully"
    echo "  Total workers: $(echo "$DASHBOARD_RESPONSE" | jq -r '.total_workers')"
    echo "  Available workers: $(echo "$DASHBOARD_RESPONSE" | jq -r '.available_workers')"
    echo "  Busy workers: $(echo "$DASHBOARD_RESPONSE" | jq -r '.busy_workers')"

    ACTIVE_TESTS=$(echo "$DASHBOARD_RESPONSE" | jq -r '.active_tests | length')
    echo "  Active tests: $ACTIVE_TESTS"

    if [ "$ACTIVE_TESTS" -gt 0 ]; then
        echo "  Active test names:"
        echo "$DASHBOARD_RESPONSE" | jq -r '.active_tests[] | "    - " + .test_name + " (" + .status + ")"'
    fi
else
    echo "❌ Failed to get dashboard data"
    echo "$DASHBOARD_RESPONSE"
fi
echo ""

# Test 8: Test aggregation trigger
echo "9. Testing manual aggregation trigger..."
if [ "$VEGETA_TEST_ID" != "null" ] && [ -n "$VEGETA_TEST_ID" ]; then
    echo "Triggering aggregation for test: $VEGETA_TEST_ID"
    AGGREGATION_RESPONSE=$(curl -s -X POST "$API_BASE/tests/$VEGETA_TEST_ID/aggregate" \
      -H "Authorization: Bearer $TOKEN")

    if echo "$AGGREGATION_RESPONSE" | jq -e '.message' > /dev/null; then
        echo "✅ Aggregation triggered successfully"
        echo "  Message: $(echo "$AGGREGATION_RESPONSE" | jq -r '.message')"
    else
        echo "❌ Failed to trigger aggregation"
        echo "$AGGREGATION_RESPONSE"
    fi
else
    echo "❌ No test ID available for aggregation testing"
fi
echo ""

# Test 9: Error handling tests
echo "=== Testing Error Handling ==="
echo ""

echo "10. Testing error cases..."

# Test invalid distribution mode
echo "Testing invalid distribution mode..."
INVALID_RESPONSE=$(curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Invalid Distribution Test",
    "duration_seconds": "5s",
    "rate_per_second": 10,
    "worker_count": 2,
    "rate_distribution": "invalid_mode",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }')

if echo "$INVALID_RESPONSE" | grep -q "error\|Error\|invalid\|Invalid"; then
    echo "✅ Invalid distribution mode properly rejected"
else
    echo "❌ Invalid distribution mode not properly handled"
fi

# Test weighted without weights
echo "Testing weighted distribution without weights..."
NO_WEIGHTS_RESPONSE=$(curl -s -X POST "$API_BASE/test/submit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Weighted Without Weights Test",
    "duration_seconds": "5s",
    "rate_per_second": 10,
    "worker_count": 3,
    "rate_distribution": "weighted",
    "targets_base64": "'"$TARGETS_BASE64"'",
    "vegeta_payload_json": "{}"
  }')

if echo "$NO_WEIGHTS_RESPONSE" | grep -q "error\|Error\|weight\|Weight"; then
    echo "✅ Weighted distribution without weights properly rejected"
else
    echo "❌ Weighted distribution without weights not properly handled"
fi

# Test non-existent test detail
echo "Testing non-existent test detail..."
FAKE_TEST_ID="non-existent-test-id"
NOT_FOUND_RESPONSE=$(curl -s -X GET "$API_BASE/tests/$FAKE_TEST_ID" \
  -H "Authorization: Bearer $TOKEN")

if echo "$NOT_FOUND_RESPONSE" | grep -q "not found\|Not found\|404"; then
    echo "✅ Non-existent test properly handled"
else
    echo "❌ Non-existent test not properly handled"
fi
echo ""

# Final summary
echo "=== Test Summary ==="
echo ""
echo "✅ Enhanced features tested:"
echo "  - Pagination functionality"
echo "  - Test detail retrieval"
echo "  - Test replay functionality"
echo "  - Enhanced rate distribution modes (weighted, ramped)"
echo "  - Enhanced Vegeta options"
echo "  - Dashboard functionality"
echo "  - Manual aggregation trigger"
echo "  - Error handling and validation"
echo ""
echo "🎉 Enhanced distributed load tester testing completed!"
echo ""
echo "Next steps:"
echo "1. Open http://localhost:8080 in your browser"
echo "2. Login with admin/password"
echo "3. Navigate through the enhanced UI:"
echo "   - Dashboard: View worker status and active tests"
echo "   - New Test: Create tests with enhanced options"
echo "   - Test History: Browse paginated history with replay and detail view"
echo "4. Test the new features:"
echo "   - Click 'View' on any test to see detailed results and charts"
echo "   - Click 'Replay' to duplicate and rerun tests"
echo "   - Try different rate distribution modes"
echo "   - Configure advanced Vegeta options"
echo ""
echo "Enhanced UI Features:"
echo "  📊 Interactive charts and visualizations"
echo "  📱 Responsive design with Tailwind CSS"
echo "  🔄 Real-time dashboard updates via WebSocket"
echo "  📖 Paginated test history"
echo "  🎯 Detailed test result analysis"
echo "  ⚡ One-click test replay functionality"
echo "  🎛️ Advanced configuration options"
