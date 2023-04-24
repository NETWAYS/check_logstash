package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"
)

func TestPipeline_ConnectionRefused(t *testing.T) {

	cmd := exec.Command("go", "run", "../main.go", "pipeline", "--port", "9999", "--inflight-events-warn", "10", "--inflight-events-crit", "20")
	out, _ := cmd.CombinedOutput()

	actual := string(out)
	expected := "UNKNOWN - Get \"http://localhost:9999/"

	if !strings.Contains(actual, expected) {
		t.Error("\nActual: ", actual, "\nExpected: ", expected)
	}
}

type PipelineTest struct {
	name     string
	server   *httptest.Server
	args     []string
	expected string
}

func TestPipelineCmd_Logstash7(t *testing.T) {
	tests := []PipelineTest{
		{
			name: "pipeline-missing-flags",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			})),
			args:     []string{"run", "../main.go", "pipeline"},
			expected: "UNKNOWN - required flag",
		},
		{
			name: "pipeline-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"7.17.8","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":null,"last_error":null,"last_failure_timestamp":null,"failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "--inflight-events-warn", "200", "--inflight-events-crit", "500"},
			expected: "OK - Inflight events",
		},
		{
			name: "pipeline-perfdata",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"7.17.8","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":null,"last_error":null,"last_failure_timestamp":null,"failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "--inflight-events-warn", "200", "--inflight-events-crit", "500"},
			expected: "| pipelines.localhost-input.events.in=100c pipelines.localhost-input.events.out=50c inflight_events_localhost-input=50;200;500 pipelines.localhost-input.reloads.failures=0 pipelines.localhost-input.reloads.successes=0",
		},
		{
			name: "pipeline-missing",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"path": "/_node/stats/pipelines/foo","status": 404,"error": {"message": "Not Found"}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "--inflight-events-warn", "10", "--inflight-events-crit", "20", "--pipeline", "foo"},
			expected: "UNKNOWN - Could not get",
		},
		{
			name: "pipeline-inflight-warn",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"7.17.8","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":null,"last_error":null,"last_failure_timestamp":null,"failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "--inflight-events-warn", "25", "--inflight-events-crit", "60"},
			expected: "WARNING - Inflight events",
		},
		{
			name: "pipeline-inflight-crit",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"7.17.8","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":null,"last_error":null,"last_failure_timestamp":null,"failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "--inflight-events-warn", "25", "--inflight-events-crit", "49"},
			expected: "CRITICAL - Inflight events",
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

func TestPipelineCmd_Logstash8(t *testing.T) {
	tests := []PipelineTest{
		{
			name: "pipeline-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"8.6","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":null,"last_error":null,"last_failure_timestamp":null,"failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "--inflight-events-warn", "200", "--inflight-events-crit", "500"},
			expected: "OK - Inflight events",
		},
		{
			name: "pipeline-reload-no-success",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"8.6","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":"","last_error":null,"last_failure_timestamp":"2020-10-11T01:10:10.11Z","failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "reload"},
			expected: "UNKNOWN - Configuration reload status unknown",
		},
		{
			name: "pipeline-reload-not-timestamp",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"8.6","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":"not a timestamp","last_error":null,"last_failure_timestamp":"no time for you","failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "reload"},
			expected: "[UNKNOWN] Configuration reload for pipeline",
		},
		{
			name: "pipeline-reload-failed",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"8.6","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":"2020-10-11T01:10:10.11Z","last_error":null,"last_failure_timestamp":"2021-10-11T01:10:10.11Z","failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "reload"},
			expected: "[CRITICAL] Configuration reload for pipeline localhost-input",
		},
		{
			name: "pipeline-reload-ok",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"host":"localhost","version":"8.6","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"localhost-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":"2020-10-11T01:10:10.11Z","last_error":null,"last_failure_timestamp":"2019-10-11T01:10:10.11Z","failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "reload"},
			expected: "[OK] Configuration successfully reloaded for pipeline localhost-input",
		},
		{
			name: "pipeline-reload-missing",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"path": "/_node/stats/pipelines/foo","status": 404,"error": {"message": "Not Found"}}`))
			})),
			args:     []string{"run", "../main.go", "pipeline", "reload", "--pipeline", "foo"},
			expected: "UNKNOWN - Could not get",
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
