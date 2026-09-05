package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

const (
	defaultAPIBase   = "http://localhost:8080"
	defaultFrontend  = "http://localhost:3000"
	maxPollAttempts  = 120
	pollInterval     = 1 * time.Second
	defaultTimeout   = 120 * time.Second
)

type Config struct {
	APIBase      string
	Concurrency  int
	Iterations   int
	Warmup       int
	Cleanup      bool
	Timeout      time.Duration
	OutputJSON   string
	OutputCSV    string
}

type Result struct {
	Iteration    int       `json:"iteration"`
	SandboxID    string    `json:"sandbox_id"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	DurationMs   float64   `json:"duration_ms"`
	APILatencyMs float64   `json:"api_latency_ms"`
	State        string    `json:"state"`
	Error        string    `json:"error,omitempty"`
}

type Summary struct {
	TotalRuns      int     `json:"total_runs"`
	Successful     int     `json:"successful"`
	Failed         int     `json:"failed"`
	TimeoutCount   int     `json:"timeout_count"`
	ThroughputRPS  float64 `json:"throughput_rps"`
	P50Ms          float64 `json:"p50_ms"`
	P95Ms          float64 `json:"p95_ms"`
	P99Ms          float64 `json:"p99_ms"`
	MeanMs         float64 `json:"mean_ms"`
	StdDevMs       float64 `json:"stddev_ms"`
	APILatencyP50  float64 `json:"api_latency_p50_ms"`
	APILatencyP95  float64 `json:"api_latency_p95_ms"`
}

func main() {
	cfg := parseFlags()

	slog.Info("berth benchmark harness starting",
		"concurrency", cfg.Concurrency,
		"iterations", cfg.Iterations,
		"warmup", cfg.Warmup,
		"cleanup", cfg.Cleanup,
	)

	client := newHTTPClient()

	// Authenticate
	if err := authenticate(client, cfg.APIBase); err != nil {
		slog.Error("authentication failed", "error", err)
		os.Exit(1)
	}
	slog.Info("authenticated via dev-login")

	projectID, err := getProjectID(client, cfg.APIBase)
	if err != nil {
		slog.Error("failed to get project id", "error", err)
		os.Exit(1)
	}
	slog.Info("fetched default project", "project_id", projectID)

	// Warmup
	if cfg.Warmup > 0 {
		slog.Info("warming up", "count", cfg.Warmup)
		runPhase(client, cfg, cfg.Warmup, true, projectID)
	}

	// Benchmark
	slog.Info("starting benchmark")
	results := runPhase(client, cfg, cfg.Iterations, false, projectID)

	// Compute stats
	summary := computeSummary(results, cfg)

	// Output
	printTable(summary, results)
	exportJSON(cfg.OutputJSON, results, summary)
	exportCSV(cfg.OutputCSV, results)

	// Cleanup
	if cfg.Cleanup {
		slog.Info("cleaning up sandboxes")
		cleanup(client, cfg.APIBase, results)
	}

	slog.Info("benchmark complete")
}

func parseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.APIBase, "api", defaultAPIBase, "Berth API base URL")
	flag.IntVar(&cfg.Concurrency, "c", 10, "Concurrent goroutines")
	flag.IntVar(&cfg.Iterations, "n", 50, "Total sandbox creations")
	flag.IntVar(&cfg.Warmup, "warmup", 3, "Warmup iterations (not counted)")
	flag.BoolVar(&cfg.Cleanup, "cleanup", true, "Delete sandboxes after test")
	flag.DurationVar(&cfg.Timeout, "timeout", defaultTimeout, "Max wait for sandbox to be RUNNING")
	flag.StringVar(&cfg.OutputJSON, "json", "bench-results.json", "Raw results JSON file")
	flag.StringVar(&cfg.OutputCSV, "csv", "bench-results.csv", "Raw results CSV file")
	flag.Parse()
	return cfg
}

func newHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 30 * time.Second, Jar: jar}
}

func authenticate(client *http.Client, base string) error {
	req, _ := http.NewRequest("GET", base+"/api/auth/dev-login", nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dev-login failed: %s", string(body))
	}
	return nil
}

func getProjectID(client *http.Client, base string) (string, error) {
	req, _ := http.NewRequest("GET", base+"/api/projects", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Projects) == 0 {
		return "", fmt.Errorf("no projects found")
	}
	return result.Projects[0].ID, nil
}

func runPhase(client *http.Client, cfg Config, count int, isWarmup bool, projectID string) []Result {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Concurrency)
	results := make([]Result, count)
	var mu sync.Mutex

	// Resource sampling goroutine
	stopSampling := make(chan struct{})
	if !isWarmup {
		go sampleResources(stopSampling)
	}

	start := time.Now()
	for i := 0; i < count; i++ {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(iter int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			r := createAndWait(client, cfg, iter, projectID)
			mu.Lock()
			results[iter] = r
			mu.Unlock()

			if !isWarmup {
				slog.Info("iteration complete",
					"iter", iter+1,
					"total", count,
					"duration_ms", fmt.Sprintf("%.0f", r.DurationMs),
					"state", r.State,
				)
			}
		}(i)
	}
	wg.Wait()
	close(stopSampling)

	if !isWarmup {
		elapsed := time.Since(start).Seconds()
		slog.Info("phase complete", "elapsed_sec", fmt.Sprintf("%.2f", elapsed))
	}
	return results
}

func createAndWait(client *http.Client, cfg Config, iter int, projectID string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	name := fmt.Sprintf("bench-%s-%d", uuid.New().String()[:8], iter)
	start := time.Now()

	// 1. POST /api/environments
	reqBody := fmt.Sprintf(`{"name":"%s","git_url":"https://github.com/octocat/Hello-World","git_branch":"master","project_id":"%s"}`, name, projectID)
	req, _ := http.NewRequestWithContext(ctx, "POST", cfg.APIBase+"/api/environments", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	apiDone := time.Now()
	apiLatency := apiDone.Sub(start).Milliseconds()

	if err != nil {
		return Result{Iteration: iter, StartTime: start, APILatencyMs: float64(apiLatency), Error: err.Error(), State: "REQUEST_FAILED"}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return Result{Iteration: iter, StartTime: start, APILatencyMs: float64(apiLatency), Error: string(body), State: "HTTP_ERROR"}
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return Result{Iteration: iter, StartTime: start, APILatencyMs: float64(apiLatency), Error: "parse id failed", State: "PARSE_ERROR"}
	}

	// 2. Poll until RUNNING or timeout
	state, pollErr := pollSandboxState(ctx, client, cfg.APIBase, created.ID)

	end := time.Now()
	return Result{
		Iteration:    iter,
		SandboxID:    created.ID,
		StartTime:    start,
		EndTime:      end,
		DurationMs:   float64(end.Sub(start).Milliseconds()),
		APILatencyMs: float64(apiLatency),
		State:        state,
		Error:        pollErr,
	}
}

func pollSandboxState(ctx context.Context, client *http.Client, base, id string) (string, string) {
	url := base + "/api/environments/" + id
	for i := 0; i < maxPollAttempts; i++ {
		select {
		case <-ctx.Done():
			return "TIMEOUT", "polling timed out"
		default:
		}

		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var env struct {
			State string `json:"state"`
		}
		json.Unmarshal(body, &env)

		if env.State == "RUNNING" {
			return "RUNNING", ""
		}
		if env.State == "FAILED" {
			return "FAILED", "sandbox entered failed state"
		}

		time.Sleep(pollInterval)
	}
	return "TIMEOUT", "max poll attempts reached"
}

func sampleResources(stop <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			cpuPct, _ := cpu.Percent(0, false)
			v, _ := mem.VirtualMemory()
			if len(cpuPct) > 0 {
				slog.Info("resource sample", "cpu_pct", fmt.Sprintf("%.1f", cpuPct[0]), "mem_pct", fmt.Sprintf("%.1f", v.UsedPercent))
			}
		}
	}
}

func computeSummary(results []Result, cfg Config) Summary {
	var durations []float64
	var apiLatencies []float64
	var successful, failed, timeouts int

	for _, r := range results {
		if r.State == "RUNNING" {
			durations = append(durations, r.DurationMs)
			successful++
		} else if r.State == "TIMEOUT" {
			timeouts++
			failed++
		} else {
			failed++
		}
		apiLatencies = append(apiLatencies, r.APILatencyMs)
	}

	s := Summary{
		TotalRuns:    len(results),
		Successful:   successful,
		Failed:       failed,
		TimeoutCount: timeouts,
	}

	if len(durations) > 0 {
		sort.Float64s(durations)
		s.P50Ms = percentile(durations, 0.50)
		s.P95Ms = percentile(durations, 0.95)
		s.P99Ms = percentile(durations, 0.99)
		s.MeanMs = mean(durations)
		s.StdDevMs = stddev(durations, s.MeanMs)
	}

	if len(apiLatencies) > 0 {
		sort.Float64s(apiLatencies)
		s.APILatencyP50 = percentile(apiLatencies, 0.50)
		s.APILatencyP95 = percentile(apiLatencies, 0.95)
	}

	elapsed := cfg.Timeout // rough upper bound for throughput calc
	s.ThroughputRPS = float64(successful) / elapsed.Seconds()

	return s
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func stddev(vals []float64, mean float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		d := v - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(vals)-1))
}

func printTable(s Summary, results []Result) {
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           BERTH COLD-START BENCHMARK RESULTS               ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Total Runs:        %-40d ║\n", s.TotalRuns)
	fmt.Printf("║ Successful:        %-40d ║\n", s.Successful)
	fmt.Printf("║ Failed:            %-40d ║\n", s.Failed)
	fmt.Printf("║ Timeouts:          %-40d ║\n", s.TimeoutCount)
	fmt.Printf("║ Throughput:        %-40.2f ║\n", s.ThroughputRPS)
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Cold-Start p50:    %-40.0f ║\n", s.P50Ms)
	fmt.Printf("║ Cold-Start p95:    %-40.0f ║\n", s.P95Ms)
	fmt.Printf("║ Cold-Start p99:    %-40.0f ║\n", s.P99Ms)
	fmt.Printf("║ Cold-Start Mean:   %-40.0f ║\n", s.MeanMs)
	fmt.Printf("║ Cold-Start StdDev: %-40.0f ║\n", s.StdDevMs)
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ API Latency p50:   %-40.0f ║\n", s.APILatencyP50)
	fmt.Printf("║ API Latency p95:   %-40.0f ║\n", s.APILatencyP95)
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

func exportJSON(path string, results []Result, summary Summary) {
	type output struct {
		Results []Result `json:"results"`
		Summary Summary  `json:"summary"`
	}
	f, _ := os.Create(path)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(output{Results: results, Summary: summary})
	slog.Info("results exported", "file", path)
}

func exportCSV(path string, results []Result) {
	f, _ := os.Create(path)
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{"iteration", "sandbox_id", "start_time", "end_time", "duration_ms", "api_latency_ms", "state", "error"})
	for _, r := range results {
		w.Write([]string{
			fmt.Sprint(r.Iteration),
			r.SandboxID,
			r.StartTime.Format(time.RFC3339),
			r.EndTime.Format(time.RFC3339),
			fmt.Sprintf("%.0f", r.DurationMs),
			fmt.Sprintf("%.0f", r.APILatencyMs),
			r.State,
			r.Error,
		})
	}
	w.Flush()
	slog.Info("results exported", "file", path)
}

func cleanup(client *http.Client, base string, results []Result) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20)
	for _, r := range results {
		if r.SandboxID == "" {
			continue
		}
		wg.Add(1)
		semaphore <- struct{}{}
		go func(id string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			req, _ := http.NewRequest("DELETE", base+"/api/environments/"+id, nil)
			resp, err := client.Do(req)
			if err != nil {
				slog.Warn("cleanup failed", "id", id, "error", err)
				return
			}
			resp.Body.Close()
		}(r.SandboxID)
	}
	wg.Wait()
}
