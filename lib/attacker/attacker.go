package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"
)

var N int

type Result struct {
	successCount int
	failureCount int
	totalCount   int
	latency      []float64
}

func main() {
	// Command-line flags
	url := flag.String("url", "http://localhost:8080", "Target URL")
	numRequests := flag.Int("n", 100, "Total number of requests")
	concurrent := flag.Int("c", 10, "Number of concurrent requests")
	// keepAlive := flag.Bool("k", false, "Keep connections alive")

	flag.Parse()

	if *url == "" {
		fmt.Println("Please provide a valid URL using the -url flag.")
		return
	}

	fmt.Printf("Starting benchmark with %d total requests, %d concurrent, targeting %s\n", *numRequests, *concurrent, *url)

	// HTTP client setup
	transport := &http.Transport{
		MaxIdleConns:        *concurrent,
		MaxIdleConnsPerHost: *concurrent,
		IdleConnTimeout:     1 * time.Second,
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Transport: transport,
	}

	// Channels and wait group
	reqCh := make(chan struct{}, *numRequests)
	resultChan := make(chan Result)

	// Start time
	startTime := time.Now()

	// Goroutine to generate requests
	go func() {
		for i := 0; i < *numRequests; i++ {
			reqCh <- struct{}{}
		}
		close(reqCh)
	}()

	for i := 0; i < *concurrent; i++ {
		go func() {
			var res Result
			for range reqCh {
				start := time.Now()
				resp, err := client.Get(*url)
				if err != nil || resp.StatusCode != http.StatusOK {
					res.failureCount++
					if err != nil {
						fmt.Printf("Error: %v\n", err)
					} else {
						fmt.Printf("Non-OK HTTP status: %s\n", resp.Status)
					}
					continue
				}
				end := time.Now()
				res.latency = append(res.latency, float64(end.Sub(start).Milliseconds()))
				res.successCount++
				resp.Body.Close()
			}
			resultChan <- res
		}()
	}

	successCount := 0
	failureCount := 0
	totalRequests := 0
	var latencies []float64

	for i := 0; i < *concurrent; i++ {
		res := <-resultChan
		successCount += res.successCount
		failureCount += res.failureCount
		latencies = append(latencies, res.latency...)
	}
	totalRequests = failureCount + successCount
	endTime := time.Now()

	fmt.Println("\nBenchmark Results:")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Successful Requests: %d\n", successCount)
	fmt.Printf("Failed Requests: %d\n", failureCount)
	fmt.Printf("Mean latency: %0.2f ms\n", calculateMean(latencies))
	fmt.Printf("99p of latency: %0.2f ms\n", calculatePercentile(latencies, 99))
	fmt.Printf("Duration: %f s\n", endTime.Sub(startTime).Seconds())
	fmt.Printf("Transactions per Second: %.2f\n", float64(totalRequests)/endTime.Sub(startTime).Seconds())
}

// CalculatePercentile calculates the p-th percentile of a sorted array of values using rounding up.
func calculatePercentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		panic("data slice is empty")
	}

	// Sort the data
	sort.Float64s(data)

	// Calculate the index for the p-th percentile, rounded up
	index := math.Ceil((p/100.0)*float64(len(data))) - 1
	return data[int(index)]
}

func calculateMean(data []float64) float64 {
	var sum float64
	for _, d := range data {
		sum += d
	}

	return sum / float64(len(data))
}
