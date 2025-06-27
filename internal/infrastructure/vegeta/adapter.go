// internal/infrastructure/vegeta/adapter.go
package vegeta

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	lib "github.com/tsenart/vegeta/v12/lib" // Corrected import path
)

// VegetaAdapter implements the domain.VegetaExecutor interface.
type VegetaAdapter struct{}

// NewVegetaAdapter creates a new Vegeta adapter.
func NewVegetaAdapter() *VegetaAdapter {
	return &VegetaAdapter{}
}

// Attack executes a Vegeta load test based on the provided configuration.
func (va *VegetaAdapter) Attack(ctx context.Context, vegetaPayloadJSON, durationStr string, rate uint64, targetsBase64 string) (*domain.TestResult, error) {
	// 1. Parse targets
	decodedTargets, err := base64.StdEncoding.DecodeString(targetsBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode targets from base64: %w", err)
	}

	targetsReader := bytes.NewReader(decodedTargets)
	var targets []lib.Target // Use lib.Target
	// Use standard json.NewDecoder to parse targets, as vegeta.NewJSONDecoder is for results.
	err = json.NewDecoder(targetsReader).Decode(&targets)
	if err != nil {
		// Fallback to simple plain text targets if JSON parsing fails
		log.Printf("Warning: Failed to decode targets as JSON: %v. Attempting to parse as plain text.", err)
		targetsReader = bytes.NewReader(decodedTargets) // Reset reader
		targeter := lib.NewStaticTargeter(targets...)   // Corrected: use lib.NewStaticTargeter
		for {
			var target lib.Target
			if err := targeter(&target); err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("failed to parse targets as plain text: %w", err)
			}
			targets = append(targets, target)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse targets as plain text or JSON: %w", err)
		}
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets found in the provided targets data")
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
	attacker := lib.NewAttacker() // Use lib.NewAttacker
	if vegetaPayloadJSON != "" {
		var attackOptions map[string]interface{}
		err = json.Unmarshal([]byte(vegetaPayloadJSON), &attackOptions)
		if err != nil {
			log.Printf("Warning: Failed to unmarshal vegetaPayloadJSON: %v. Using default attacker options.", err)
			// Continue with default attacker if payload is invalid
		} else {
			// Apply specific attacker options if they exist in the payload
			if timeout, ok := attackOptions["timeout"].(float64); ok {
				attacker = lib.NewAttacker(lib.Client(&http.Client{Timeout: time.Duration(timeout) * time.Second}))
			}
			if redirects, ok := attackOptions["redirects"].(float64); ok {
				attacker = lib.NewAttacker(lib.Client(&http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						if len(via) >= int(redirects) {
							return http.ErrUseLastResponse
						}
						return nil
					},
				}))
			}
			// Add more options as needed (connections, http2, keepalive, etc.)
			// Note: Converting map[string]interface{} to direct vegeta.Attacker options can be complex.
			// For a comprehensive solution, you might need reflection or specific struct mapping.
			// For this example, we'll just handle a few common ones.
		}
	}

	// 5. Start the attack
	var m lib.Metrics // Use lib.Metrics directly
	results := attacker.Attack(lib.NewStaticTargeter(targets...), attackRate, duration, "Load Test")
	for res := range results {
		m.Add(res)
	}
	m.Close() // Important: Close the metrics collector to finalize calculations

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
		StatusCodes:       mapToString(convertStatusCodes(m.StatusCodes)),
	}

	return testResult, nil
}

// mapToString converts a map[string]uint64 to a JSON string.
func mapToString(m map[string]uint64) string {
	b, err := json.Marshal(m)
	if err != nil {
		log.Printf("Error marshaling map to JSON: %v", err)
		return "{}"
	}
	return string(b)
}

// convertStatusCodes converts a map[string]int to a map[string]uint64.
func convertStatusCodes(statusCodes map[string]int) map[string]uint64 {
	converted := make(map[string]uint64)
	for key, value := range statusCodes {
		converted[key] = uint64(value)
	}
	return converted
}
