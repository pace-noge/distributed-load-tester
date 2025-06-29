# 📋 Quick Reference - Distributed Load Tester

## 🚀 Quick Start Checklist

- [ ] Open browser to `http://localhost:8080`
- [ ] Login with credentials
- [ ] Check that workers are available (Dashboard)
- [ ] Go to "New Test"
- [ ] Fill in basic info and submit

## ⚙️ Test Configuration Examples

### Basic API Test
```
Name: "API Health Check"
URL: "https://api.example.com/health"
Method: GET
Rate: 50 req/s
Duration: 2m
Workers: 3
Distribution: Shared
```

### Authentication Test
```
Name: "Login Test"
URL: "https://api.example.com/login"
Method: POST
Rate: 25 req/s
Duration: 5m
Headers: {"Content-Type": "application/json"}
Body: {"username": "test", "password": "test123"}
```

### High Load Stress Test
```
Name: "Stress Test"
URL: "https://api.example.com/users"
Method: GET
Rate: 200 req/s
Duration: 3m
Workers: 3
Distribution: Same (for 600 total req/s)
```

## 📊 Rate Distribution Quick Guide

| Mode | Description | Use When |
|------|-------------|----------|
| **Shared** | Split rate evenly | Testing specific throughput |
| **Same** | Each worker sends full rate | Stress testing, finding limits |
| **Weighted** | Custom per worker | Geographic simulation |
| **Ramped** | Gradually increasing | Auto-scaling tests |
| **Burst** | Front-loaded | Traffic spike simulation |

## 🎯 Rate Calculation Examples

### Shared Distribution
```
Rate: 300 req/s + 3 workers = 100 req/s per worker
Total load: 300 req/s
```

### Same Distribution
```
Rate: 100 req/s + 3 workers = 100 req/s per worker
Total load: 300 req/s
```

## 📈 Reading Results

### ✅ Good Performance
- Success Rate: > 95%
- Avg Latency: < 200ms
- P95 Latency: < 500ms
- Errors: < 1%

### ⚠️ Performance Issues
- Success Rate: 80-95%
- Avg Latency: 200-1000ms
- P95 Latency: 500-2000ms
- Errors: 1-10%

### ❌ System Overload
- Success Rate: < 80%
- Avg Latency: > 1000ms
- P95 Latency: > 2000ms
- Errors: > 10%

## 🔧 Common Duration Formats

| Format | Duration |
|--------|----------|
| `30s` | 30 seconds |
| `2m` | 2 minutes |
| `5m` | 5 minutes |
| `1h` | 1 hour |
| `90s` | 1.5 minutes |

## 🚨 Troubleshooting Quick Fixes

| Problem | Quick Fix |
|---------|-----------|
| No workers available | Check worker status, restart if needed |
| Tests stuck in RUNNING | Wait 10s for auto-cleanup |
| All requests failing | Check URL, auth headers |
| Very slow responses | Reduce rate, check target service |
| Results vary widely | Run longer tests, check for caching |

## 💡 Best Practice Checklist

### Before Testing
- [ ] Get permission to test the target system
- [ ] Test during off-peak hours
- [ ] Start with low load (10-50 req/s)
- [ ] Have monitoring ready

### During Testing
- [ ] Monitor system resources
- [ ] Watch for error patterns
- [ ] Stop if system shows distress
- [ ] Document any issues

### After Testing
- [ ] Analyze all metrics, not just success rate
- [ ] Correlate with system monitoring
- [ ] Document findings
- [ ] Plan improvements

## 📞 Common HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| 200 | Success | ✅ Good |
| 401 | Unauthorized | Check auth headers |
| 403 | Forbidden | Check permissions |
| 404 | Not Found | Check URL |
| 429 | Rate Limited | Reduce load |
| 500 | Server Error | Check target service |
| 502 | Bad Gateway | Check upstream |
| 503 | Unavailable | Service down |
| 504 | Timeout | Reduce load or check network |

---

**Need help?** Check the full [USER_GUIDE.md](USER_GUIDE.md) for detailed explanations!
