package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"
)

func TestHealth_ConnectionRefused(t *testing.T) {
	cmd := exec.Command("go", "run", "../main.go", "health", "--port", "9999")
	out, _ := cmd.CombinedOutput()

	actual := string(out)
	expected := "[UNKNOWN] - Get \"http://localhost:9999/"

	if !strings.Contains(actual, expected) {
		t.Error("\nActual: ", actual, "\nExpected: ", expected)
	}
}

func TestHealth_ConnectionRefusedCritical(t *testing.T) {
	cmd := exec.Command("go", "run", "../main.go", "health", "--port", "9999", "--unreachable-state", "2")
	out, _ := cmd.CombinedOutput()

	actual := string(out)
	expected := "[CRITICAL] - Get \"http://localhost:9999/"

	if !strings.Contains(actual, expected) {
		t.Error("\nActual: ", actual, "\nExpected: ", expected)
	}

	cmd = exec.Command("go", "run", "../main.go", "health", "--port", "9999", "--unreachable-state", "-123")
	out, _ = cmd.CombinedOutput()

	actual = string(out)
	expected = "[UNKNOWN] - Get \"http://localhost:9999/"

	if !strings.Contains(actual, expected) {
		t.Error("\nActual: ", actual, "\nExpected: ", expected)
	}
}

type HealthTest struct {
	name     string
	server   *httptest.Server
	args     []string
	expected string
}

func TestHealthCmd_Logstash6(t *testing.T) {
	tests := []HealthTest{
		{
			name: "version-error",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"logstash","version":"foo"}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "[UNKNOWN] - could not determine version",
		},
		{
			name: "health-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"logstash","version":"6.8.23","http_address":"0.0.0.0:9600","id":"123","name":"logstash","jvm":{"threads":{"count":1,"peak_count":2},"mem":{},"gc":{},"uptime_in_millis":123},"process":{},"events":{},"pipelines":{"main":{}},"reloads":{"failures":0,"successes":0},"os":{}}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "[OK] - Logstash is healthy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.server.Close()

			// We need the random Port extracted
			u, _ := url.Parse(test.server.URL)
			cmd := exec.Command("go", append(test.args, "--port", u.Port())...)
			out, _ := cmd.CombinedOutput()

			actual := string(out)

			if !strings.Contains(actual, test.expected) {
				t.Error("\nActual: ", actual, "\nExpected: ", test.expected)
			}

		})
	}
}

func TestHealthCmd_Logstash7(t *testing.T) {
	tests := []HealthTest{
		{
			name: "health-api-error",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"foo": "bar"}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "[UNKNOWN] - could not determine status",
		},
		{
			name: "health-bearer-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token := r.Header.Get("Authorization")
				if token == "Bearer secret" {
					// Just for testing, this is now how to handle tokens properly
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 120,"peak_open_file_descriptors": 120,"max_file_descriptors":16384,"cpu":{"percent": 1}}}`))
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`The Authorization header wasn't set`))
			})),
			args:     []string{"run", "../main.go", "--bearer", "secret", "health"},
			expected: "[OK] - Logstash is healthy",
		},
		{
			name: "health-bearer-unauthorized",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token := r.Header.Get("Authorization")
				if token == "Bearer right-token" {
					// Just for testing, this is now how to handle BasicAuth properly
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{}`))
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`Access Denied!`))
			})),
			args:     []string{"run", "../main.go", "--bearer", "wrong-token", "health"},
			expected: "[UNKNOWN] - could not get ",
		},
		{
			name: "health-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 120,"peak_open_file_descriptors": 120,"max_file_descriptors":16384,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "[OK] - Logstash is healthy",
		},
		{
			name: "health-perfdata",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 120,"peak_open_file_descriptors": 120,"max_file_descriptors":16384,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "| process.cpu.percent=1%;100;100;0;100 jvm.mem.heap_used_percent=20%;70;80;0;100 jvm.threads.count=50;;;;0 process.open_file_descriptors=120;100;100;0;16384",
		},
		{
			name: "health-red",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"red","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 120,"peak_open_file_descriptors": 120,"max_file_descriptors":16384,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "[CRITICAL] - Logstash is unhealthy",
		},
		{
			name: "health-filedesc-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 1,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--file-descriptor-threshold-crit", "50"},
			expected: "[OK] - Logstash is healthy",
		},
		{
			name: "health-filedesc-warn",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 45,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--file-descriptor-threshold-warn", "40", "--file-descriptor-threshold-crit", "50"},
			expected: "[WARNING] Open file descriptors at 45.00%",
		},
		{
			name: "health-filedesc-crit",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--file-descriptor-threshold-warn", "40", "--file-descriptor-threshold-crit", "50"},
			expected: "[CRITICAL] Open file descriptors at 51.00%",
		},
		{
			name: "health-heapuse-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":50}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--heap-usage-threshold-warn", "50"},
			expected: "[OK] - Logstash is healthy",
		},
		{
			name: "health-heapuse-warn",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":45}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--heap-usage-threshold-warn", "40", "--heap-usage-threshold-crit", "50"},
			expected: "[WARNING] Heap usage at 45.00%",
		},
		{
			name: "health-heapuse-crit",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":51}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--heap-usage-threshold-warn", "40", "--heap-usage-threshold-crit", "50"},
			expected: "[CRITICAL] Heap usage at 51.00%",
		},
		{
			name: "health-cpuuse-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":50}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 50}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--cpu-usage-threshold-warn", "50"},
			expected: "[OK] - Logstash is healthy",
		},
		{
			name: "health-cpuuse-warn",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":50}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 45}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--cpu-usage-threshold-warn", "40", "--cpu-usage-threshold-crit", "50"},
			expected: "[WARNING] CPU usage at 45.00%",
		},
		{
			name: "health-cpuuse-crit",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":50}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 51}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--cpu-usage-threshold-warn", "40", "--cpu-usage-threshold-crit", "50"},
			expected: "[CRITICAL] CPU usage at 51.00%",
		},
		{
			name: "health-cpu-heap-worst-state",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":55}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 45}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--cpu-usage-threshold-warn", "40", "--heap-usage-threshold-crit", "50"},
			expected: "[CRITICAL] - Logstash is unhealthy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.server.Close()

			// We need the random Port extracted
			u, _ := url.Parse(test.server.URL)
			cmd := exec.Command("go", append(test.args, "--port", u.Port())...)
			out, _ := cmd.CombinedOutput()

			actual := string(out)

			if !strings.Contains(actual, test.expected) {
				t.Error("\nActual: ", actual, "\nExpected: ", test.expected)
			}

		})
	}
}

func TestHealthCmd_Logstash8(t *testing.T) {
	tests := []HealthTest{
		{
			name: "health-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"8.6","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 120,"peak_open_file_descriptors": 120,"max_file_descriptors":16384,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "[OK] - Logstash is healthy",
		},
		{
			name: "health-perfdata",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"8.6","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 120,"peak_open_file_descriptors": 120,"max_file_descriptors":16384,"cpu":{"percent": 1}}}`))
			})),
			args:     []string{"run", "../main.go", "health"},
			expected: "| process.cpu.percent=1%;100;100;0;100 jvm.mem.heap_used_percent=20%;70;80;0;100 jvm.threads.count=50;;;;0 process.open_file_descriptors=120;100;100;0;16384",
		},
		{
			name: "health-cpu-heap-worst-state",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"test","version":"8.6","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":55}},"process":{"open_file_descriptors": 51,"peak_open_file_descriptors": 50,"max_file_descriptors":100,"cpu":{"percent": 45}}}`))
			})),
			args:     []string{"run", "../main.go", "health", "--cpu-usage-threshold-warn", "40", "--heap-usage-threshold-crit", "50"},
			expected: "[CRITICAL] - Logstash is unhealthy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.server.Close()

			// We need the random Port extracted
			u, _ := url.Parse(test.server.URL)
			cmd := exec.Command("go", append(test.args, "--port", u.Port())...)
			out, _ := cmd.CombinedOutput()

			actual := string(out)

			if !strings.Contains(actual, test.expected) {
				t.Error("\nActual: ", actual, "\nExpected: ", test.expected)
			}

		})
	}
}
