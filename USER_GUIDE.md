# 🚀 Distributed Load Tester - User Guide

Welcome to the Distributed Load Tester! This guide will help you understand how to use the load testing system effectively, whether you're a developer, QA engineer, or DevOps professional.

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Understanding the Dashboard](#understanding-the-dashboard)
3. [Creating Your First Test](#creating-your-first-test)
4. [Rate Distribution Modes](#rate-distribution-modes)
5. [Understanding Test Results](#understanding-test-results)
6. [Common Use Cases](#common-use-cases)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

---

## 🏁 Quick Start

### What is this system?

The Distributed Load Tester is a web-based tool that helps you test how your applications perform under load. It can simulate many users hitting your API or website simultaneously to see how it responds.

### Key Features

- **🌐 Web Dashboard**: Easy-to-use interface accessible from any browser
- **⚡ Distributed Testing**: Multiple workers spread the load for realistic testing
- **📊 Real-time Monitoring**: Watch your tests progress in real-time
- **📈 Detailed Analytics**: Comprehensive reports with latency, throughput, and error rates
- **🎯 Flexible Targeting**: Test HTTP APIs, websites, or any HTTP endpoint

### Accessing the System

1. Open your web browser
2. Navigate to: `http://localhost:8080` (or your server's address)
3. Login with your credentials (default: `admin` / `password`)

---

## 📊 Understanding the Dashboard

### Main Dashboard Overview

When you first log in, you'll see:

- **Worker Status**: Shows how many testing workers are available
- **Active Tests**: Tests currently running
- **Recent Tests**: History of completed tests
- **System Health**: Overall system status

### Worker Information

- **Total Workers**: Total number of testing agents available
- **Available**: Workers ready to accept new tests
- **Busy**: Workers currently executing tests
- **Worker Names**: Each worker has a unique, memorable name (e.g., "SwiftRedFalcon-7X2K")

---

## 🎯 Creating Your First Test

### Step 1: Navigate to New Test

Click **"New Test"** from the navigation menu.

### Step 2: Basic Test Configuration

Fill in the basic information:

#### **Test Name**
```
Example: "Login API Load Test"
```
Choose a descriptive name that helps you identify the test later.

#### **Target URL**
```
Example: "https://api.myapp.com/login"
```
The endpoint you want to test. Can be any HTTP/HTTPS URL.

#### **Duration**
```
Example: "30s" (30 seconds)
         "2m"  (2 minutes)
         "5m"  (5 minutes)
```
How long the test should run. Use formats like `30s`, `2m`, `1h`.

#### **Rate (Requests per Second)**
```
Example: 100 (sends 100 requests per second)
```
How many requests to send per second. Start with lower numbers and gradually increase.

#### **Number of Workers**
```
Example: 3 (uses all available workers)
```
How many workers to use. More workers = more distributed load.

### Step 3: HTTP Configuration

#### **HTTP Method**
Choose from:
- **GET**: For retrieving data (most common for testing)
- **POST**: For creating/submitting data
- **PUT**: For updating data
- **DELETE**: For removing data

#### **Request Headers** (Optional)
Add headers if needed:
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer your-token-here"
}
```

#### **Request Body** (For POST/PUT requests)
```json
{
  "username": "testuser",
  "password": "testpass"
}
```

### Step 4: Rate Distribution

Choose how requests are distributed across workers:

- **Shared**: Total rate split evenly (Recommended for most tests)
- **Same**: Each worker sends the full rate (Higher total load)
- **Weighted**: Custom distribution per worker
- **Ramped**: Gradually increasing load
- **Burst**: Front-loaded distribution

### Step 5: Submit Test

Click **"Submit Test"** to start your load test!

---

## ⚖️ Rate Distribution Modes

Understanding how requests are distributed across workers is crucial for effective testing.

### 🔄 Shared Distribution (Recommended)

**How it works**: Total rate is divided evenly among workers
```
Rate: 300 req/s
Workers: 3
Result: Each worker sends 100 req/s
Total: 300 req/s
```

**Best for**:
- Testing specific throughput limits
- Simulating exact user loads
- Most common testing scenarios

**Example Use Case**: "I want to see if my API can handle exactly 500 requests per second"

### 📈 Same Distribution

**How it works**: Each worker sends the full rate
```
Rate: 100 req/s
Workers: 3
Result: Each worker sends 100 req/s
Total: 300 req/s
```

**Best for**:
- Stress testing
- Finding breaking points
- High-load scenarios

**Example Use Case**: "I want to overwhelm my system to find its limits"

### ⚡ Weighted Distribution

**How it works**: Custom weights determine load per worker
```
Rate: 300 req/s
Workers: 3
Weights: [2.0, 1.0, 1.0]
Result: Worker 1: 150 req/s, Worker 2: 75 req/s, Worker 3: 75 req/s
```

**Best for**:
- Simulating uneven geographic distribution
- Testing specific scaling scenarios
- Advanced testing patterns

### 📊 Ramped Distribution

**How it works**: Load increases gradually across workers
```
Rate: 300 req/s
Workers: 3
Result: Worker 1: 50 req/s, Worker 2: 100 req/s, Worker 3: 150 req/s
```

**Best for**:
- Gradual load increase simulation
- Testing auto-scaling behavior
- Warm-up scenarios

### 💥 Burst Distribution

**How it works**: Higher load on initial workers
```
Rate: 300 req/s
Workers: 3
Result: Worker 1: 150 req/s, Worker 2: 100 req/s, Worker 3: 50 req/s
```

**Best for**:
- Simulating traffic spikes
- Testing initial load handling
- Peak load scenarios

---

## 📈 Understanding Test Results

### Test Status

- **🟢 COMPLETED**: All workers finished successfully
- **🟡 PARTIALLY_FAILED**: Some workers completed, others failed
- **🔴 FAILED**: All workers failed or test was stopped
- **🔵 RUNNING**: Test is currently executing

### Key Metrics

#### **Throughput**
- **Total Requests**: How many requests were sent
- **Completed Requests**: How many got responses
- **Requests/Second**: Average rate achieved

#### **Latency**
- **Average Latency**: Mean response time
- **P95 Latency**: 95% of requests were faster than this
- **P99 Latency**: 99% of requests were faster than this

#### **Success Rate**
- **Success Rate**: Percentage of successful responses (2xx status codes)
- **Error Rate**: Percentage of failed requests

#### **Status Codes**
- **2xx**: Successful responses
- **4xx**: Client errors (bad requests, unauthorized, etc.)
- **5xx**: Server errors (internal server errors, timeouts, etc.)

### Reading the Results

#### ✅ Good Results
```
Success Rate: 99.5%
Average Latency: 45ms
P95 Latency: 120ms
Status Codes: 99% 200 OK
```
Your system is handling the load well!

#### ⚠️ Warning Signs
```
Success Rate: 85%
Average Latency: 500ms
P95 Latency: 2000ms
Status Codes: 15% 5xx errors
```
Your system is struggling with the load.

#### ❌ System Overload
```
Success Rate: 30%
Average Latency: 5000ms
P95 Latency: 10000ms
Status Codes: 70% timeouts/5xx
```
The load is too high for your system.

---

## 🎮 Common Use Cases

### 1. API Endpoint Testing

**Scenario**: Test if your REST API can handle expected traffic

**Configuration**:
```
Test Name: "User Profile API Test"
URL: "https://api.myapp.com/user/profile"
Method: GET
Rate: 50 req/s
Duration: 2m
Workers: 3
Distribution: Shared
Headers: {"Authorization": "Bearer token"}
```

**What to look for**:
- Success rate > 95%
- Average latency < 200ms
- No 5xx errors

### 2. Login System Load Test

**Scenario**: Test authentication under load

**Configuration**:
```
Test Name: "Login Load Test"
URL: "https://api.myapp.com/auth/login"
Method: POST
Rate: 25 req/s
Duration: 5m
Workers: 3
Distribution: Shared
Body: {"username": "test@example.com", "password": "password"}
```

**What to look for**:
- Consistent response times
- No authentication failures
- Database performance

### 3. E-commerce Checkout Stress Test

**Scenario**: Test checkout process during high traffic (Black Friday)

**Configuration**:
```
Test Name: "Checkout Stress Test"
URL: "https://shop.myapp.com/api/checkout"
Method: POST
Rate: 200 req/s
Duration: 10m
Workers: 3
Distribution: Same (for stress testing)
```

**What to look for**:
- Payment processing reliability
- Inventory management accuracy
- System breaking point

### 4. CDN Performance Test

**Scenario**: Test content delivery performance

**Configuration**:
```
Test Name: "Image CDN Test"
URL: "https://cdn.myapp.com/images/hero.jpg"
Method: GET
Rate: 500 req/s
Duration: 3m
Workers: 3
Distribution: Shared
```

**What to look for**:
- Fast response times globally
- Cache hit rates
- Geographic performance differences

### 5. Gradual Load Increase

**Scenario**: Slowly increase load to find capacity limits

**Run multiple tests with increasing rates**:
```
Test 1: 50 req/s  (baseline)
Test 2: 100 req/s (2x increase)
Test 3: 200 req/s (4x increase)
Test 4: 400 req/s (8x increase)
```

**What to look for**:
- At what point do errors start appearing?
- How does latency increase with load?
- When does the system become unstable?

---

## 💡 Best Practices

### Before Testing

1. **🎯 Define Clear Goals**
   - What are you trying to measure?
   - What's your expected capacity?
   - What constitutes success/failure?

2. **📋 Test in Staging First**
   - Never run load tests on production without planning
   - Use staging environments that mirror production
   - Ensure you have permission to run tests

3. **📊 Baseline Performance**
   - Run a low-load test first to establish baseline
   - Record normal response times and error rates
   - Document current system capacity

### During Testing

1. **👀 Monitor System Resources**
   - Watch CPU, memory, and database performance
   - Monitor application logs for errors
   - Keep an eye on dependent services

2. **📈 Start Small, Scale Up**
   - Begin with low rates (10-50 req/s)
   - Gradually increase load
   - Stop if you see system degradation

3. **⏱️ Appropriate Test Duration**
   - Short tests (30s-2m): Quick functionality checks
   - Medium tests (5-10m): Sustained load testing
   - Long tests (30m+): Endurance and memory leak testing

### After Testing

1. **📊 Analyze Results Thoroughly**
   - Look beyond just success rates
   - Examine latency distributions
   - Identify patterns in failures

2. **🔍 Correlate with System Metrics**
   - Match load test results with server monitoring
   - Identify bottlenecks (CPU, memory, database, network)
   - Document findings for future reference

3. **🔧 Plan Improvements**
   - Prioritize performance issues found
   - Set capacity planning goals
   - Schedule follow-up tests after improvements

### Load Testing Etiquette

1. **⚠️ Get Permission**
   - Always get approval before testing external services
   - Respect rate limits and terms of service
   - Consider the impact on other users

2. **🕒 Test During Off-Peak Hours**
   - Run tests when usage is naturally low
   - Avoid business-critical hours
   - Coordinate with your team

3. **📢 Communicate Tests**
   - Notify your team when running tests
   - Document test schedules
   - Share results with stakeholders

---

## 🔧 Troubleshooting

### Common Issues

#### Problem: "No workers available"
**Symptoms**: Tests fail to start, error message about no workers
**Solutions**:
- Check that workers are running and connected
- Verify worker status in the dashboard
- Restart the worker services if needed

#### Problem: "All requests failing"
**Symptoms**: 0% success rate, all requests return errors
**Solutions**:
- Check the target URL is correct and accessible
- Verify authentication headers/tokens
- Ensure the target service is running

#### Problem: "Tests stuck in RUNNING status"
**Symptoms**: Test doesn't complete, shows as running indefinitely
**Solutions**:
- Wait for automatic cleanup (runs every 10 seconds)
- Check worker logs for errors
- Restart the master service if needed

#### Problem: "Slow response times"
**Symptoms**: Very high latency, tests timing out
**Solutions**:
- Reduce the request rate
- Check network connectivity
- Verify target service performance
- Consider using fewer concurrent workers

#### Problem: "Inconsistent results"
**Symptoms**: Results vary dramatically between test runs
**Solutions**:
- Run longer tests for more stable averages
- Check for external factors (network, other traffic)
- Ensure test conditions are consistent
- Look for auto-scaling or caching effects

### Performance Tuning Tips

#### For Better Accuracy
- Use consistent test durations (2-5 minutes minimum)
- Run tests multiple times and average results
- Control for external variables
- Test during similar conditions each time

#### For Higher Throughput
- Increase number of workers
- Use "Same" distribution mode for maximum load
- Optimize network connectivity between workers and targets
- Ensure workers have sufficient resources

#### For Realistic Simulation
- Use "Shared" distribution mode
- Include realistic request payloads
- Add appropriate authentication headers
- Simulate real user behavior patterns

---

## 🎓 Learning More

### Understanding HTTP Status Codes

- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **400 Bad Request**: Invalid request format
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Access denied
- **404 Not Found**: Resource doesn't exist
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server-side error
- **502 Bad Gateway**: Upstream server error
- **503 Service Unavailable**: Service temporarily down
- **504 Gateway Timeout**: Request timeout

### Performance Metrics Explained

#### **Latency vs Throughput**
- **Latency**: How fast individual requests complete
- **Throughput**: How many requests complete per second
- These can be inversely related - higher throughput might mean higher latency

#### **Percentiles (P95, P99)**
- **P95**: 95% of requests were faster than this time
- **P99**: 99% of requests were faster than this time
- More representative of user experience than averages

#### **Concurrency vs Rate**
- **Rate**: Requests per second
- **Concurrency**: Number of simultaneous requests
- Higher rate doesn't always mean higher concurrency

---

## 📞 Support

### Getting Help

1. **Check the Dashboard**: Look for error messages or system status
2. **Review This Guide**: Common issues are covered in troubleshooting
3. **Check Logs**: Application logs contain detailed error information
4. **Contact Your Team**: Reach out to the development or DevOps team

### Reporting Issues

When reporting problems, include:
- What you were trying to do
- Exact error messages
- Test configuration used
- Screenshots of the dashboard
- Timestamp when the issue occurred

---

**Happy Load Testing! 🚀**

*Remember: The goal of load testing is to find issues before your users do. Start small, be methodical, and always test with purpose.*
