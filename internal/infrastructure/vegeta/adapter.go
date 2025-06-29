// internal/infrastructure/vegeta/adapter.go
package vegeta

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	lib "github.com/tsenart/vegeta/v12/lib" // Corrected import path
	"golang.org/x/net/http2"
)

// VegetaAdapter implements the domain.VegetaExecutor interface.
type VegetaAdapter struct{}

// NewVegetaAdapter creates a new Vegeta adapter.
func NewVegetaAdapter() *VegetaAdapter {
	return &VegetaAdapter{}
}

// Attack executes a Vegeta load test based on the provided configuration.
func (va *VegetaAdapter) Attack(ctx context.Context, vegetaPayloadJSON, durationStr string, rate uint64, targetsBase64 string) (*domain.TestResult, error) {
	log.Printf("Starting Vegeta attack with duration=%s, rate=%d, targetsBase64 length=%d", durationStr, rate, len(targetsBase64))

	// 1. Parse targets
	decodedTargets, err := base64.StdEncoding.DecodeString(targetsBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode targets from base64: %w", err)
	}

	log.Printf("Decoded targets: %s", string(decodedTargets))

	targetsReader := bytes.NewReader(decodedTargets)
	var targets []lib.Target // Use lib.Target
	// Use standard json.NewDecoder to parse targets, as vegeta.NewJSONDecoder is for results.
	err = json.NewDecoder(targetsReader).Decode(&targets)
	if err != nil {
		// Fallback to simple plain text targets if JSON parsing fails
		log.Printf("Warning: Failed to decode targets as JSON: %v. Attempting to parse as plain text.", err)
		targetsReader = bytes.NewReader(decodedTargets) // Reset reader

		// Parse as plain text - each line should be a URL
		lines := bytes.Split(decodedTargets, []byte("\n"))
		for _, line := range lines {
			lineStr := string(bytes.TrimSpace(line))
			if lineStr == "" {
				continue // Skip empty lines
			}
			// Create a basic GET target for each URL
			target := lib.Target{
				Method: "GET",
				URL:    lineStr,
				Header: make(http.Header),
			}
			targets = append(targets, target)
		}
	}

	// Ensure we have at least one target
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets found in the provided targets data")
	}

	log.Printf("Parsed %d targets successfully", len(targets))
	for i, target := range targets {
		log.Printf("Target %d: %s %s", i, target.Method, target.URL)
	}

	// 2. Parse duration
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid duration string: %w", err)
	}

	// 3. Create rate
	var attackRate lib.Rate // Use lib.Rate
	if rate > 0 {
		attackRate = lib.Rate{Freq: int(rate), Per: time.Second}
	} else {
		return nil, fmt.Errorf("rate per second must be greater than 0")
	}

	// 4. Configure attacker options (from vegetaPayloadJSON)
	var attackOptions map[string]interface{}
	if vegetaPayloadJSON != "" {
		err = json.Unmarshal([]byte(vegetaPayloadJSON), &attackOptions)
		if err != nil {
			log.Printf("Warning: Failed to unmarshal vegetaPayloadJSON: %v. Using default attacker options.", err)
			attackOptions = make(map[string]interface{})
		}
	} else {
		attackOptions = make(map[string]interface{})
	}

	// Build HTTP client with enhanced options
	client := &http.Client{}

	// Timeout configuration
	timeout := 30.0 // Default 30 seconds
	if val, ok := attackOptions["timeout"].(float64); ok {
		timeout = val
	}
	client.Timeout = time.Duration(timeout) * time.Second

	// Redirect configuration
	maxRedirects := 10 // Default 10 redirects
	if val, ok := attackOptions["redirects"].(float64); ok {
		maxRedirects = int(val)
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return http.ErrUseLastResponse
		}
		return nil
	}

	// TLS configuration
	tlsConfig := &tls.Config{}
	if val, ok := attackOptions["insecure"].(bool); ok && val {
		tlsConfig.InsecureSkipVerify = true
	}

	// HTTP/2 configuration
	http2Enabled := true // Default enable HTTP/2
	if val, ok := attackOptions["http2"].(bool); ok {
		http2Enabled = val
	}

	// Connection pooling
	maxConnections := 100 // Default max connections
	if val, ok := attackOptions["connections"].(float64); ok {
		maxConnections = int(val)
	}

	// Keep-alive configuration
	keepAlive := true // Default enable keep-alive
	if val, ok := attackOptions["keepalive"].(bool); ok {
		keepAlive = val
	}

	// Build transport with enhanced options
	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        maxConnections,
		MaxIdleConnsPerHost: maxConnections / 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   !keepAlive,
	}

	// Configure HTTP/2
	if http2Enabled {
		err = http2.ConfigureTransport(transport)
		if err != nil {
			log.Printf("Warning: Failed to configure HTTP/2: %v", err)
		}
	} else {
		transport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	}

	client.Transport = transport

	// Create attacker with configured client
	attacker := lib.NewAttacker(lib.Client(client))

	// Apply additional options if supported
	// Note: Some options like workers, max_body might need different approaches
	// depending on the vegeta version

	// 5. Start the attack
	log.Printf("Starting Vegeta attack: rate=%v, duration=%v, targets=%d", attackRate, duration, len(targets))
	var m lib.Metrics // Use lib.Metrics directly
	results := attacker.Attack(lib.NewStaticTargeter(targets...), attackRate, duration, "Load Test")
	for res := range results {
		m.Add(res)
	}
	m.Close() // Important: Close the metrics collector to finalize calculations
	log.Printf("Vegeta attack completed")

	// 6. Convert Vegeta metrics to domain.TestResult
	testResult := &domain.TestResult{
		Metric: func() []byte {
			b, err := json.Marshal(m)
			if err != nil {
				log.Printf("Error marshaling metrics to JSON: %v", err)
				return []byte("{}")
			}
			return b
		}(), // Store full Vegeta metric output
		TotalRequests:     int64(m.Requests),
		CompletedRequests: int64(m.Requests), // Total number of requests
		DurationMs:        m.Duration.Milliseconds(),
		SuccessRate:       m.Success,
		AverageLatencyMs:  float64(m.Latencies.Mean.Milliseconds()),
		P95LatencyMs:      float64(m.Latencies.P95.Milliseconds()),
		StatusCodes:       m.StatusCodes,
	}

	return testResult, nil
}
