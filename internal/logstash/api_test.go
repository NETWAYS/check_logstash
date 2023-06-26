package logstash

import (
	"encoding/json"
	"testing"
)

func TestUmarshallPipelineFlow(t *testing.T) {

	j := `{"host":"foobar","version":"8.7.1","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"ansible-input":{"flow":{"queue_backpressure":{"current":10,"last_1_minute":0,"lifetime":2.503e-05},"output_throughput":{"current":0,"last_1_minute":0.344,"lifetime":0.7051},"input_throughput":{"current":10,"last_1_minute":0.5734,"lifetime":1.089},"worker_concurrency":{"current":0.0001815,"last_1_minute":0.0009501,"lifetime":0.003384},"filter_throughput":{"current":0,"last_1_minute":0.5734,"lifetime":1.089}},"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":null,"last_error":null,"last_failure_timestamp":null,"failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`

	var pl Pipeline
	err := json.Unmarshal([]byte(j), &pl)

	if err != nil {
		t.Error(err)
	}

	if pl.Pipelines["ansible-input"].Flow.QueueBackpressure.Current != 10 {
		t.Error("\nActual: ", pl.Pipelines["ansible-input"].Flow.QueueBackpressure.Current, "\nExpected: ", "10")
	}

	if pl.Pipelines["ansible-input"].Flow.InputThroughput.Current != 10 {
		t.Error("\nActual: ", pl.Pipelines["ansible-input"].Flow.InputThroughput.Current, "\nExpected: ", "10")
	}
}

func TestUmarshallPipeline(t *testing.T) {

	j := `{"host":"foobar","version":"7.17.8","http_address":"127.0.0.1:9600","id":"4","name":"test","ephemeral_id":"5","status":"green","snapshot":false,"pipeline":{"workers":2,"batch_size":125,"batch_delay":50},"pipelines":{"ansible-input":{"events":{"filtered":0,"duration_in_millis":0,"queue_push_duration_in_millis":0,"out":50,"in":100},"plugins":{"inputs":[{"id":"b","name":"beats","events":{"queue_push_duration_in_millis":0,"out":0}}],"codecs":[{"id":"plain","name":"plain","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}},{"id":"json","name":"json","decode":{"writes_in":0,"duration_in_millis":0,"out":0},"encode":{"writes_in":0,"duration_in_millis":0}}],"filters":[],"outputs":[{"id":"f","name":"redis","events":{"duration_in_millis":18,"out":50,"in":100}}]},"reloads":{"successes":0,"last_success_timestamp":null,"last_error":null,"last_failure_timestamp":null,"failures":0},"queue":{"type":"memory","events_count":0,"queue_size_in_bytes":0,"max_queue_size_in_bytes":0},"hash":"f","ephemeral_id":"f"}}}`

	var pl Pipeline
	err := json.Unmarshal([]byte(j), &pl)

	if err != nil {
		t.Error(err)
	}

	if pl.Host != "foobar" {
		t.Error("\nActual: ", pl.Host, "\nExpected: ", "foobar")
	}

}

func TestUmarshallStat(t *testing.T) {

	j := `{"host":"foobar","version":"7.17.8","status":"green","jvm":{"threads":{"count":50,"peak_count":51},"mem":{"heap_used_percent":20}},"process":{"open_file_descriptors": 120,"peak_open_file_descriptors": 120,"max_file_descriptors":16384,"cpu":{"percent": 1}}}`

	var st Stat
	err := json.Unmarshal([]byte(j), &st)

	if err != nil {
		t.Error(err)
	}

	if st.Host != "foobar" {
		t.Error("\nActual: ", st.Host, "\nExpected: ", "foobar")
	}

}
