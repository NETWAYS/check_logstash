package logstash

// https://www.elastic.co/guide/en/logstash/current/node-stats-api.html

type Pipeline struct {
	Host      string `json:"host"`
	Pipelines map[string]struct {
		Reloads struct {
			Successes int `json:"successes"`
			Failures  int `json:"failures"`
		} `json:"reloads"`
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
	Host    string  `json:"host"`
	Version string  `json:"version"`
	Status  string  `json:"status"`
	Process Process `json:"process"`
	Jvm     JVM     `json:"jvm"`
}
