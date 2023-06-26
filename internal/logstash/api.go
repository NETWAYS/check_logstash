package logstash

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// https://www.elastic.co/guide/en/logstash/current/node-stats-api.html

type Pipeline struct {
	Host      string `json:"host"`
	Pipelines map[string]struct {
		Reloads struct {
			LastSuccessTime string `json:"last_success_timestamp"`
			LastFailureTime string `json:"last_failure_timestamp"`
			Successes       int    `json:"successes"`
			Failures        int    `json:"failures"`
		} `json:"reloads"`
		Flow struct {
			QueueBackpressure FlowMetric `json:"queue_backpressure"`
			OutputThroughput  FlowMetric `json:"output_throughput"`
			InputThroughput   FlowMetric `json:"input_throughput"`
			FilterThroughput  FlowMetric `json:"filter_throughput"`
		} `json:"flow"`
		Queue struct {
			Type                string `json:"type"`
			EventsCount         int    `json:"events_count"`
			QueueSizeInBytes    int    `json:"queue_size_in_bytes"`
			MaxQueueSizeInBytes int    `json:"max_queue_size_in_bytes"`
		} `json:"queue"`
		Events struct {
			Filtered          int `json:"filtered"`
			Duration          int `json:"duration"`
			QueuePushDuration int `json:"queue_push_duration_in_millis"`
			In                int `json:"in"`
			Out               int `json:"out"`
		} `json:"events"`
	} `json:"pipelines"`
}

type FlowMetric struct {
	Current     float64 `json:"current"`
	Last1Minute float64 `json:"last_1_minute"`
	Lifetime    float64 `json:"lifetime"`
}

type Process struct {
	MaxFileDescriptors  float64 `json:"max_file_descriptors"`
	OpenFileDescriptors float64 `json:"open_file_descriptors"`
	CPU                 struct {
		Percent float64 `json:"percent"`
	} `json:"cpu"`
}

type JVM struct {
	Mem struct {
		HeapUsedPercent float64 `json:"heap_used_percent"`
	}
	Threads struct {
		Count     int `json:"count"`
		PeakCount int `json:"peak_count"`
	}
}

type Stat struct {
	Host         string  `json:"host"`
	Version      string  `json:"version"`
	Status       string  `json:"status"`
	Process      Process `json:"process"`
	Jvm          JVM     `json:"jvm"`
	MajorVersion int
}

// Custom Unmarshal since we might want to add or parse
// further fields in the future. This is simpler to extend and
// to test here than during the CheckPlugin logic.
func (s *Stat) UnmarshalJSON(b []byte) error {
	type Temp Stat

	t := (*Temp)(s)

	err := json.Unmarshal(b, t)

	if err != nil {
		return err
	}

	// Could also use some semver package,
	// but decided against the depedency
	if s.Version != "" {
		v := strings.Split(s.Version, ".")
		majorVersion, convErr := strconv.Atoi(v[0])

		if convErr != nil {
			return fmt.Errorf("Could not determine version")
		}

		s.MajorVersion = majorVersion
	}

	return nil
}
