package web

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/weka/amg-utils/tools/amg-qad/internal/storage"
)

// Server represents the web server
type Server struct {
	port    int
	storage *storage.Storage
	server  *http.Server
}

// New creates a new web server
func New(port int, store *storage.Storage) *Server {
	return &Server{
		port:    port,
		storage: store,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	router := mux.NewRouter()

	router.HandleFunc("/", s.handleDashboard).Methods("GET")
	router.HandleFunc("/api/results", s.handleAPIResults).Methods("GET")

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: router,
	}

	go func() {
		log.Printf("Web server starting on port %d", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Web server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the web server
func (s *Server) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}
	}
}

// handleDashboard serves the main dashboard page
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	results, err := s.storage.GetLastResults(10)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get results: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>AMG-QAD Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1000px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .stats { display: flex; justify-content: space-around; margin-bottom: 30px; }
        .stat-card { background: #f8f9fa; padding: 15px; border-radius: 5px; text-align: center; min-width: 120px; }
        .stat-number { font-size: 24px; font-weight: bold; color: #007bff; }
        .stat-label { color: #666; font-size: 14px; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: bold; }
        .status-passed { color: #28a745; font-weight: bold; }
        .status-failed { color: #dc3545; font-weight: bold; }
        .timestamp { font-family: monospace; }
        .duration { font-family: monospace; color: #666; }
        .no-results { text-align: center; color: #666; padding: 40px; }
        .refresh { margin-bottom: 20px; }
        .refresh button { background: #007bff; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
        .refresh button:hover { background: #0056b3; }
        .test-suite-row { cursor: pointer; background: #f8f9fa; }
        .test-suite-row:hover { background: #e9ecef; }
        .individual-tests { display: none; }
        .individual-tests.expanded { display: table-row-group; }
        .individual-test-row { background: #ffffff; border-left: 4px solid #007bff; }
        .individual-test-row td { padding-left: 30px; font-size: 0.9em; color: #666; }
        .expand-icon { margin-right: 8px; transition: transform 0.2s; }
        .expand-icon.expanded { transform: rotate(90deg); }
    </style>
    <script>
        function refreshData() {
            location.reload();
        }
        
        function toggleTests(index) {
            const testsRow = document.getElementById('tests-' + index);
            const icon = document.getElementById('icon-' + index);
            
            if (testsRow.classList.contains('expanded')) {
                testsRow.classList.remove('expanded');
                icon.textContent = '[+]';
                icon.classList.remove('expanded');
            } else {
                testsRow.classList.add('expanded');
                icon.textContent = '[-]';
                icon.classList.add('expanded');
            }
        }
        
        // Auto-refresh every 30 seconds
        setInterval(refreshData, 30000);
    </script>
</head>
<body>
    <div class="container">
        <h1>AMG Quality Assurance Dashboard</h1>
        
        <div class="stats">
            <div class="stat-card">
                <div class="stat-number">{{.TotalTests}}</div>
                <div class="stat-label">Total Tests</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{.PassedTests}}</div>
                <div class="stat-label">Passed</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{.FailedTests}}</div>
                <div class="stat-label">Failed</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{.SuccessRate}}%</div>
                <div class="stat-label">Success Rate</div>
            </div>
        </div>
        
        <div class="refresh">
            <button onclick="refreshData()">Refresh</button>
            <span style="color: #666; margin-left: 10px;">Last updated: {{.LastUpdated}}</span>
        </div>
        
        {{if .Results}}
        <table>
            <thead>
                <tr>
                    <th>Timestamp</th>
                    <th>Status</th>
                    <th>Duration</th>
                    <th>Parameters</th>
                </tr>
            </thead>
            <tbody>
                {{range $index, $suite := .Results}}
                <tr class="test-suite-row" onclick="toggleTests({{$index}})">
                    <td class="timestamp">{{.FormattedTime}}</td>
                    <td class="{{if .Passed}}status-passed{{else}}status-failed{{end}}">
                        {{if .Passed}}PASSED{{else}}FAILED{{end}}
                    </td>
                    <td class="duration">{{.Duration}}</td>
                    <td>
                        <span class="expand-icon" id="icon-{{$index}}">[+]</span>
                        {{.Parameters}} ({{.TestCount}} tests)
                    </td>
                </tr>
                {{if .IndividualTests}}
                <tr class="individual-tests" id="tests-{{$index}}">
                    <td colspan="4" style="padding: 0;">
                        <table style="width: 100%; margin: 0;">
                            {{range .IndividualTests}}
                            <tr class="individual-test-row">
                                <td class="timestamp" style="width: 25%;"></td>
                                <td class="{{if .Passed}}status-passed{{else}}status-failed{{end}}" style="width: 25%;">
                                    {{if .Passed}}PASSED{{else}}FAILED{{end}}
                                </td>
                                <td class="duration" style="width: 25%;">{{.Duration}}</td>
                                <td style="width: 25%;">{{.Name}}</td>
                            </tr>
                            {{end}}
                        </table>
                    </td>
                </tr>
                {{end}}
                {{end}}
            </tbody>
        </table>
        {{else}}
        <div class="no-results">
            <p>No test results available yet.</p>
            <p>Tests will appear here once they start running.</p>
        </div>
        {{end}}
    </div>
</body>
</html>`

	data := s.prepareDashboardData(results)

	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// prepareDashboardData prepares data for the dashboard template
func (s *Server) prepareDashboardData(results []storage.TestResult) map[string]interface{} {
	data := make(map[string]interface{})

	// Filter to only show test suite results (not individual tests)
	var suiteResults []storage.TestResult
	for _, result := range results {
		if strings.HasPrefix(result.Parameters, "test_suite_") {
			suiteResults = append(suiteResults, result)
		}
	}

	// Process suite results for display
	var processedResults []map[string]interface{}
	passed := 0
	total := len(suiteResults)

	for _, result := range suiteResults {
		// Prepare individual tests for expansion
		var individualTests []map[string]interface{}
		for _, test := range result.Tests {
			individualTest := map[string]interface{}{
				"Name":     test.Name,
				"Passed":   test.Passed,
				"Duration": test.Duration,
				"Logs":     test.Logs,
			}
			individualTests = append(individualTests, individualTest)
		}

		processedResult := map[string]interface{}{
			"FormattedTime":   result.Timestamp.Format("2006-01-02 15:04:05"),
			"Passed":          result.Passed,
			"Duration":        result.Duration,
			"Parameters":      result.Parameters,
			"TestCount":       len(result.Tests),
			"IndividualTests": individualTests,
		}
		processedResults = append(processedResults, processedResult)

		if result.Passed {
			passed++
		}
	}

	successRate := 0
	if total > 0 {
		successRate = (passed * 100) / total
	}

	data["Results"] = processedResults
	data["TotalTests"] = total
	data["PassedTests"] = passed
	data["FailedTests"] = total - passed
	data["SuccessRate"] = successRate
	data["LastUpdated"] = time.Now().Format("15:04:05")

	return data
}

// handleAPIResults serves test results as JSON
func (s *Server) handleAPIResults(w http.ResponseWriter, r *http.Request) {
	limitParam := r.URL.Query().Get("limit")
	limit := 10

	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	results, err := s.storage.GetLastResults(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get results: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Simple JSON encoding without external dependencies
	if _, err := w.Write([]byte("[")); err != nil {
		return
	}
	for i, result := range results {
		if i > 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return
			}
		}
		status := "passed"
		if !result.Passed {
			status = "failed"
		}
		json := fmt.Sprintf(`{"timestamp":"%s","status":"%s","duration":"%s","parameters":"%s"}`,
			result.Timestamp.Format(time.RFC3339), status, result.Duration, result.Parameters)
		if _, err := w.Write([]byte(json)); err != nil {
			return
		}
	}
	if _, err := w.Write([]byte("]")); err != nil {
		return
	}
}
